package daemon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/tunnel/internal/routes"
	"github.com/ryangerardwilson/c3po/tunnel/internal/secrets"
)

func TestReloadOnRouteChange(t *testing.T) {
	home := t.TempDir()
	logPath := filepath.Join(home, "args")
	script := filepath.Join(home, "cloudflared")
	body := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + shellQuote(logPath) + "\nexec sleep 30\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	file := routes.Default(routes.Zone)
	if err := routes.Save(routes.Path(home), file); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var hosts []string
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Options{
			Home:        home,
			Cloudflared: script,
			Poll:        30 * time.Millisecond,
			Retry:       time.Hour,
			Load: func() (secrets.Tunnel, error) {
				return secrets.Tunnel{ID: "1bdc9e04-7e31-4589-baa9-f6caf6ae9a74", Zone: routes.Zone, Credential: []byte(`{"ok":true}`), APIToken: "t"}, nil
			},
			Ensure: func(_ context.Context, _, hostname, _ string) error {
				hosts = append(hosts, hostname)
				return nil
			},
			Logf: func(string, ...any) {},
		})
	}()
	waitLines(t, logPath, 1)
	next, _, err := routes.Upsert(file, "lab", 9090)
	if err != nil {
		t.Fatal(err)
	}
	if err := routes.Save(routes.Path(home), next); err != nil {
		t.Fatal(err)
	}
	waitLines(t, logPath, 2)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("daemon did not stop")
	}
	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) < 2 {
		t.Fatalf("starts: %q", raw)
	}
	for _, line := range lines[:2] {
		if !strings.Contains(line, "--no-autoupdate tunnel --config") || !strings.HasSuffix(line, " run") {
			t.Fatalf("args %q", line)
		}
	}
	cfg, err := os.ReadFile(filepath.Join(routes.ConfigDir(home), "cloudflared.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(cfg)
	if !strings.Contains(text, "protocol: http2") || !strings.Contains(text, "hostname: lab.ryangerardwilson.com") || !strings.Contains(text, "service: http://127.0.0.1:9090") {
		t.Fatalf("config:\n%s", text)
	}
	if len(hosts) == 0 || hosts[0] != routes.Zone {
		t.Fatalf("ensured %#v", hosts)
	}
}

func waitLines(t *testing.T, path string, n int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(path)
		if err == nil && len(strings.FieldsFunc(strings.TrimSpace(string(raw)), func(r rune) bool { return r == '\n' })) >= n {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d connector starts", n)
}

func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}
