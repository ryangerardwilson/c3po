// Package tunnel publishes ports on this laptop as subdomains of
// ryangerardwilson.com through a named Cloudflare tunnel. It is the
// app behind c3po tunnel.
package tunnel

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ryangerardwilson/c3po/internal/version"
	"github.com/ryangerardwilson/c3po/internal/xdg"
	"github.com/ryangerardwilson/c3po/tunnel/internal/daemon"
	"github.com/ryangerardwilson/c3po/tunnel/internal/dns"
	"github.com/ryangerardwilson/c3po/tunnel/internal/secrets"
	"github.com/ryangerardwilson/c3po/tunnel/internal/store"
	"github.com/ryangerardwilson/c3po/tunnel/internal/tui"
)

// Help is the full help for c3po tunnel.
const Help = `c3po tunnel
publish laptop ports as subdomains of ryangerardwilson.com

global actions:
  c3po tunnel help
    show this help
  c3po tunnel version
    print the c3po version
  c3po tunnel open
    edit routes in the terminal
  c3po tunnel start
    start the background service
  c3po tunnel stop
    stop the background service
  c3po tunnel status
    show the background service
  c3po tunnel logs
    follow the background service log
  c3po tunnel run
    run the background worker

features:
  add a subdomain and forward it to a local port
  # add <name> on <port>
  c3po tunnel add notes on 8080
  c3po tunnel open

  show or remove a route
  # list | remove <name>
  c3po tunnel list
  c3po tunnel remove notes

notes:
  A name is one label under ryangerardwilson.com, or @ for the apex.
  It forwards to 127.0.0.1 on that port. The first saved list keeps
  the apex on port 8080. Adding a name creates a proxied CNAME when
  the name has no DNS yet; a name that points somewhere else is left
  alone and not saved. Removing a route leaves DNS as it is.
  Routes live in ~/.config/c3po/tunnel/routes.json. Credentials come from
  the 1Password items CLOUDFLARE_TUNNEL_CREDENTIALS and
  CLOUDFLARE_API_TOKEN in the Root vault, read with the service
  account token in $OP_SERVICE_ACCOUNT_TOKEN_FILE
  (default ~/.config/c3po/pass/service-account-token).
  start, stop, status, and logs drive the user unit tunnel.service,
  whose ExecStart is c3po tunnel run. run keeps cloudflared
  connected and reconnects when the route list changes.

exit: 0 ok, 1 error, 2 usage
`

// Run runs c3po tunnel. args are the words after tunnel.
func Run(args []string, stdout, stderr io.Writer) int {
	return newApp(stdout, stderr).run(args)
}

// app holds the seams the commands use, so tests can replace credentials,
// DNS, systemd, the route screen, and the worker.
type app struct {
	Out       io.Writer
	Err       io.Writer
	Home      string
	TokenFile string
	Load      func() (secrets.Tunnel, error)
	Ensure    store.Ensure
	Service   func(args ...string) error
	Journal   func() error
	Open      func(home string) error
	Serve     func(ctx context.Context) error
}

func newApp(stdout, stderr io.Writer) *app {
	home := os.Getenv("HOME")
	tokenFile, err := xdg.PassToken()
	if err != nil {
		tokenFile = ""
	}
	a := &app{Out: stdout, Err: stderr, Home: home, TokenFile: tokenFile}
	a.Load = func() (secrets.Tunnel, error) {
		return secrets.Load(a.TokenFile)
	}
	a.Ensure = a.ensureDNS
	a.Open = func(home string) error {
		return tui.Run(home, a.Load, a.Ensure)
	}
	a.Serve = func(ctx context.Context) error {
		return daemon.Run(ctx, daemon.Options{Home: a.Home, TokenFile: a.TokenFile, Stdout: a.Out, Stderr: a.Err})
	}
	return a
}

func (a *app) ensureDNS(ctx context.Context, zone, hostname, tunnelID string) error {
	tunnel, err := a.Load()
	if err != nil {
		return err
	}
	client := &dns.Client{Token: tunnel.APIToken}
	return client.Ensure(ctx, zone, hostname, tunnelID)
}

type command func(a *app, rest []string) int

var commands = map[string]command{
	"help":    (*app).help,
	"-h":      (*app).help,
	"--help":  (*app).help,
	"version": (*app).version,
	"open":    (*app).open,
	"list":    (*app).listCmd,
	"add":     (*app).add,
	"remove":  (*app).remove,
	"start":   (*app).start,
	"stop":    (*app).stop,
	"status":  (*app).status,
	"logs":    (*app).logs,
	"run":     (*app).serve,
}

func (a *app) run(args []string) int {
	a.fill()
	cmd := "help"
	if len(args) > 0 {
		cmd = args[0]
	}
	rest := []string{}
	if len(args) > 1 {
		rest = args[1:]
	}
	if f, ok := commands[cmd]; ok {
		return f(a, rest)
	}
	fmt.Fprint(a.Err, paint(Help, a.Err))
	return 2
}

func (a *app) fill() {
	if a.Out == nil {
		a.Out = os.Stdout
	}
	if a.Err == nil {
		a.Err = os.Stderr
	}
	if a.Home == "" {
		a.Home = os.Getenv("HOME")
	}
}

func (a *app) help(_ []string) int {
	fmt.Fprint(a.Out, paint(Help, a.Out))
	return 0
}

func (a *app) version(_ []string) int {
	fmt.Fprintln(a.Out, version.String())
	return 0
}

func (a *app) open(_ []string) int {
	if a.Open == nil {
		fmt.Fprintln(a.Err, "tunnel open is unavailable")
		return 1
	}
	return a.report(a.Open(a.Home))
}

func (a *app) listCmd(_ []string) int {
	return a.report(a.list())
}

func (a *app) add(rest []string) int {
	if len(rest) != 3 || rest[1] != "on" {
		fmt.Fprintln(a.Err, "usage: c3po tunnel add <name> on <port>")
		return 2
	}
	_, host, err := store.Add(a.Home, rest[0], rest[2], a.Load, a.Ensure)
	if err != nil {
		return a.report(err)
	}
	fmt.Fprintf(a.Out, "added %s\n", host)
	return 0
}

func (a *app) remove(rest []string) int {
	if len(rest) != 1 {
		fmt.Fprintln(a.Err, "usage: c3po tunnel remove <name>")
		return 2
	}
	if _, err := store.Remove(a.Home, rest[0]); err != nil {
		return a.report(err)
	}
	fmt.Fprintf(a.Out, "removed %s\n", rest[0])
	return 0
}

func (a *app) serve(_ []string) int {
	if a.Serve == nil {
		fmt.Fprintln(a.Err, "tunnel run is unavailable")
		return 1
	}
	ctx, stop := signalContext()
	defer stop()
	return a.report(a.Serve(ctx))
}

// report prints err and returns its exit code.
func (a *app) report(err error) int {
	if err != nil {
		fmt.Fprintln(a.Err, err.Error())
		return 1
	}
	return 0
}

// paint dims the help on a color terminal.
func paint(text string, w io.Writer) string {
	if os.Getenv("NO_COLOR") != "" {
		return text
	}
	file, ok := w.(*os.File)
	if !ok || !isTerminal(file) {
		return text
	}
	return "\033[38;5;245m" + text + "\033[0m"
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
