package daemon

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ryangerardwilson/c3po/tunnel/internal/dns"
	"github.com/ryangerardwilson/c3po/tunnel/internal/ingress"
	"github.com/ryangerardwilson/c3po/tunnel/internal/routes"
	"github.com/ryangerardwilson/c3po/tunnel/internal/secrets"
)

// Options controls the background connector loop.
type Options struct {
	Home        string
	TokenFile   string
	Cloudflared string
	Poll        time.Duration
	Retry       time.Duration
	Load        func() (secrets.Tunnel, error)
	Ensure      func(ctx context.Context, zone, hostname, tunnelID string) error
	Logf        func(format string, args ...any)
	// Stdout and Stderr receive the connector output. They default to
	// the process streams, which the user service sends to the journal.
	Stdout io.Writer
	Stderr io.Writer
}

// Run keeps the named tunnel connected and reloads it when the route list changes.
func Run(ctx context.Context, opt Options) error {
	opt = withDefaults(opt)
	credPath := filepath.Join(routes.ConfigDir(opt.Home), "credentials.json")
	defer os.Remove(credPath)
	for ctx.Err() == nil {
		if !step(ctx, opt) {
			return nil
		}
	}
	return nil
}

func withDefaults(opt Options) Options {
	if opt.Home == "" {
		opt.Home = os.Getenv("HOME")
	}
	if opt.TokenFile == "" {
		opt.TokenFile = filepath.Join(opt.Home, ".config", "op", "service-account-token")
	}
	if opt.Cloudflared == "" {
		opt.Cloudflared = "cloudflared"
	}
	opt.Poll = positive(opt.Poll, 2*time.Second)
	opt.Retry = positive(opt.Retry, 5*time.Second)
	opt = withStreams(opt)
	if opt.Load == nil {
		tokenFile := opt.TokenFile
		opt.Load = func() (secrets.Tunnel, error) {
			return secrets.Load(tokenFile)
		}
	}
	return opt
}

func withStreams(opt Options) Options {
	if opt.Stdout == nil {
		opt.Stdout = os.Stdout
	}
	if opt.Stderr == nil {
		opt.Stderr = os.Stderr
	}
	if opt.Logf == nil {
		w := opt.Stderr
		opt.Logf = func(format string, args ...any) {
			fmt.Fprintf(w, format+"\n", args...)
		}
	}
	return opt
}

func positive(d, fallback time.Duration) time.Duration {
	if d <= 0 {
		return fallback
	}
	return d
}

// step runs one connector session. It reports false when ctx ended.
func step(ctx context.Context, opt Options) bool {
	tunnel, err := opt.Load()
	if err != nil {
		opt.Logf("tunnel: waiting for credentials")
		return sleep(ctx, opt.Retry)
	}
	reload, err := session(ctx, opt, tunnel)
	if ctx.Err() != nil {
		return false
	}
	if reload {
		opt.Logf("tunnel: routes changed, reconnecting")
		return true
	}
	if err != nil {
		opt.Logf("tunnel: %v", err)
	}
	opt.Logf("tunnel: connector exited, retrying in 5s")
	return sleep(ctx, opt.Retry)
}

func session(ctx context.Context, opt Options, tunnel secrets.Tunnel) (bool, error) {
	file, err := routes.EnsureFile(routes.Path(opt.Home))
	if err != nil {
		return false, err
	}
	if file.Zone == "" {
		file.Zone = tunnel.Zone
	}
	ensureAll(ctx, opt, tunnel, file)
	cfgPath, err := writeConfig(opt.Home, tunnel, file)
	if err != nil {
		return false, err
	}
	opt.Logf("tunnel: connecting %d routes", len(file.Routes))
	return connect(ctx, opt, file, cfgPath)
}

// ensureAll makes sure each route has DNS. Failures are logged; the
// connector still starts.
func ensureAll(ctx context.Context, opt Options, tunnel secrets.Tunnel, file routes.File) {
	ensure := opt.Ensure
	if ensure == nil && tunnel.APIToken != "" {
		client := &dns.Client{Token: tunnel.APIToken}
		ensure = client.Ensure
	}
	for _, route := range file.Routes {
		_, host, err := routes.Normalize(file.Zone, route.Name)
		if err != nil {
			opt.Logf("tunnel: %v", err)
			continue
		}
		if ensure == nil {
			continue
		}
		if err := ensure(ctx, file.Zone, host, tunnel.ID); err != nil {
			opt.Logf("tunnel: %s", err.Error())
		}
	}
}

// writeConfig writes the credential file and the cloudflared config,
// both mode 0600, and returns the config path.
func writeConfig(home string, tunnel secrets.Tunnel, file routes.File) (string, error) {
	cfgDir := routes.ConfigDir(home)
	credPath := filepath.Join(cfgDir, "credentials.json")
	cfgPath := filepath.Join(cfgDir, "cloudflared.yml")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(credPath, tunnel.Credential, 0o600); err != nil {
		return "", err
	}
	text, err := ingress.Render(tunnel.ID, credPath, file.Zone, file.Routes)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(cfgPath, []byte(text), 0o600); err != nil {
		return "", err
	}
	return cfgPath, nil
}

// connect runs cloudflared until it exits, ctx ends, or the route list
// changes. It reports true when the routes changed.
func connect(ctx context.Context, opt Options, file routes.File, cfgPath string) (bool, error) {
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(child, opt.Cloudflared, "--no-autoupdate", "tunnel", "--config", cfgPath, "run")
	cmd.WaitDelay = 2 * time.Second
	cmd.Stdout = opt.Stdout
	cmd.Stderr = opt.Stderr
	if err := cmd.Start(); err != nil {
		return false, err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	return watch(ctx, opt, file, done, cancel)
}

func watch(ctx context.Context, opt Options, file routes.File, done <-chan error, cancel context.CancelFunc) (bool, error) {
	ticker := time.NewTicker(opt.Poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			cancel()
			<-done
			return false, ctx.Err()
		case err := <-done:
			return false, err
		case <-ticker.C:
			if changed(opt, file) {
				cancel()
				<-done
				return true, nil
			}
		}
	}
}

func changed(opt Options, file routes.File) bool {
	next, err := routes.Load(routes.Path(opt.Home))
	if err != nil {
		opt.Logf("tunnel: %v", err)
		return false
	}
	return !routes.Same(next, file)
}

func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
