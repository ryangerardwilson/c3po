package tunnel

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/internal/version"
	"github.com/ryangerardwilson/c3po/tunnel/internal/dns"
	"github.com/ryangerardwilson/c3po/tunnel/internal/routes"
	"github.com/ryangerardwilson/c3po/tunnel/internal/secrets"
)

func TestHelpVersionAndAdd(t *testing.T) {
	home := t.TempDir()
	a := app{
		Home: home,
		Load: func() (secrets.Tunnel, error) {
			return secrets.Tunnel{ID: "1bdc9e04-7e31-4589-baa9-f6caf6ae9a74", Zone: routes.Zone, APIToken: "t"}, nil
		},
	}
	var ensured []string
	a.Ensure = func(_ context.Context, _, hostname, _ string) error {
		if hostname == "notes."+routes.Zone {
			return dns.ErrForeign
		}
		ensured = append(ensured, hostname)
		return nil
	}
	out, errOut, code := run(&a, "help")
	if code != 0 || !strings.Contains(out, "c3po tunnel add notes on 8080") || !strings.Contains(out, "c3po tui tunnel") || strings.Contains(out, "c3po tunnel open") {
		t.Fatalf("help %d %s", code, out)
	}
	if strings.Contains(out, "\033") {
		t.Fatal("help was painted without a terminal")
	}
	out, _, code = run(&a, "version")
	if code != 0 || strings.TrimSpace(out) != version.String() {
		t.Fatalf("version %q %d", out, code)
	}
	out, errOut, code = run(&a, "add", "Lab", "on", "9090")
	if code != 0 || !strings.Contains(out, "added lab.ryangerardwilson.com") {
		t.Fatalf("add %d %s %s", code, out, errOut)
	}
	if len(ensured) != 1 || ensured[0] != "lab.ryangerardwilson.com" {
		t.Fatalf("ensured %#v", ensured)
	}
	out, _, code = run(&a, "list")
	if code != 0 || !strings.Contains(out, "lab") || !strings.Contains(out, "9090") || !strings.Contains(out, "@") {
		t.Fatalf("list %d %s", code, out)
	}
	_, errOut, code = run(&a, "add", "notes", "on", "3000")
	if code == 0 || !strings.Contains(errOut, "points somewhere else") {
		t.Fatalf("foreign %d %s", code, errOut)
	}
	file, err := routes.Load(routes.Path(home))
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range file.Routes {
		if route.Name == "notes" {
			t.Fatal("refused route was saved")
		}
	}
	out, _, code = run(&a, "remove", "lab")
	if code != 0 || !strings.Contains(out, "removed lab") {
		t.Fatalf("remove %d %s", code, out)
	}
	_, errOut, code = run(&a, "nope")
	if code != 2 || !strings.Contains(errOut, "c3po tui tunnel") || strings.Contains(errOut, "c3po tunnel open") {
		t.Fatalf("unknown %d %s", code, errOut)
	}
	_, errOut, code = run(&a, "add", "lab")
	if code != 2 || !strings.Contains(errOut, "usage: c3po tunnel add <name> on <port>") {
		t.Fatalf("bad add %d %s", code, errOut)
	}
	_, errOut, code = run(&a, "remove")
	if code != 2 || !strings.Contains(errOut, "usage: c3po tunnel remove <name>") {
		t.Fatalf("bad remove %d %s", code, errOut)
	}
	_, errOut, code = run(&a, "remove", "ghost")
	if code != 1 || !strings.Contains(errOut, "no route named ghost") {
		t.Fatalf("missing remove %d %s", code, errOut)
	}
}

func TestAddReportsCredentialFailure(t *testing.T) {
	a := app{
		Home: t.TempDir(),
		Load: func() (secrets.Tunnel, error) {
			return secrets.Tunnel{}, errors.New("waiting for credentials")
		},
		Ensure: func(context.Context, string, string, string) error {
			t.Fatal("dns ran without credentials")
			return nil
		},
	}
	_, errOut, code := run(&a, "add", "lab", "on", "1")
	if code == 0 || !strings.Contains(errOut, "credentials") {
		t.Fatalf("%d %s", code, errOut)
	}
}

func TestRunHelpIsTheScoutHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"-h"}, {"--help"}} {
		var out, errOut bytes.Buffer
		code := Run(args, &out, &errOut)
		if code != 0 || out.String() != Help || errOut.Len() != 0 {
			t.Fatalf("%v: %d %q %q", args, code, out.String(), errOut.String())
		}
	}
	for _, line := range strings.Split(Help, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "tunnel ") {
			t.Fatalf("help names the old binary: %q", line)
		}
	}
}

func TestServiceCommandsDriveTheUserUnit(t *testing.T) {
	var calls []string
	active := true
	a := app{
		Home: t.TempDir(),
		Service: func(args ...string) error {
			calls = append(calls, strings.Join(args, " "))
			if args[0] == "--quiet" && !active {
				return errors.New("inactive")
			}
			return nil
		},
		Journal: func() error {
			calls = append(calls, "journal")
			return nil
		},
	}
	out, _, code := run(&a, "start")
	if code != 0 || out != "tunnel started\n" {
		t.Fatalf("start %d %q", code, out)
	}
	out, _, code = run(&a, "stop")
	if code != 0 || out != "tunnel stopped\n" {
		t.Fatalf("stop %d %q", code, out)
	}
	if _, _, code = run(&a, "status"); code != 0 {
		t.Fatalf("status %d", code)
	}
	if _, _, code = run(&a, "logs"); code != 0 {
		t.Fatalf("logs %d", code)
	}
	want := []string{
		"start tunnel.service",
		"--quiet is-active tunnel.service",
		"stop tunnel.service",
		"status tunnel.service --no-pager",
		"journal",
	}
	if strings.Join(calls, "|") != strings.Join(want, "|") {
		t.Fatalf("calls %#v", calls)
	}
	active = false
	_, errOut, code := run(&a, "start")
	if code != 1 || !strings.Contains(errOut, "tunnel did not start") {
		t.Fatalf("inactive start %d %q", code, errOut)
	}
}

func TestRunServesUntilTheWorkerReturns(t *testing.T) {
	served := false
	a := app{
		Home: t.TempDir(),
		Serve: func(ctx context.Context) error {
			served = ctx != nil
			return nil
		},
	}
	if _, _, code := run(&a, "run"); code != 0 || !served {
		t.Fatalf("run %d %v", code, served)
	}
	a.Serve = func(context.Context) error { return errors.New("boom") }
	if _, errOut, code := run(&a, "run"); code != 1 || !strings.Contains(errOut, "boom") {
		t.Fatalf("run error %d %q", code, errOut)
	}
}

func run(a *app, args ...string) (string, string, int) {
	var out, errOut bytes.Buffer
	a.Out = &out
	a.Err = &errOut
	code := a.run(args)
	return out.String(), errOut.String(), code
}
