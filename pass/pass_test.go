package pass

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/internal/version"
)

// isolate keeps the real op and token file out of reach.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN", "")
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN_FILE", "")
}

func TestHelpDescribesVaultSearch(t *testing.T) {
	isolate(t)
	for _, args := range [][]string{nil, {"help"}, {"-h"}, {"--help"}} {
		var out, errOut bytes.Buffer
		if rc := Run(args, &out, &errOut); rc != 0 || out.String() != Help || errOut.Len() != 0 {
			t.Fatalf("%v rc = %d", args, rc)
		}
	}
	for _, needle := range []string{
		"c3po pass <vault> <search>", "c3po pass <vault> -l", "c3po pass wiom -l",
		"c3po pass version", "wl-copy", "c3po pass <vault> --create", "c3po pass <vault> --edit",
		"c3po pass <vault> --delete", "c3po pass <vault> --show", "--template",
		"~/.config/c3po/pass/service-account-token",
	} {
		if !strings.Contains(Help, needle) {
			t.Fatalf("help missing %q", needle)
		}
	}
	for _, retired := range []string{"pass open", "pass remember csv", "password sheet", "pass copy password", "pass upgrade"} {
		if strings.Contains(Help, retired) {
			t.Fatalf("help still offers %q", retired)
		}
	}
}

func TestVersionIsTheC3poVersion(t *testing.T) {
	isolate(t)
	var out bytes.Buffer
	if rc := Run([]string{"version"}, &out, &bytes.Buffer{}); rc != 0 || out.String() != version.String()+"\n" {
		t.Fatalf("rc = %d version = %q", rc, out.String())
	}
}

func TestAVaultNamedHelpIsTheHelpWord(t *testing.T) {
	isolate(t)
	var out, errOut bytes.Buffer
	rc := Run([]string{"help", "github"}, &out, &errOut)
	if rc != 1 || errOut.String() != "Use help by itself.\n" || out.Len() != 0 {
		t.Fatalf("rc = %d stderr = %q", rc, errOut.String())
	}
}

func TestOneArgumentIsUsage(t *testing.T) {
	isolate(t)
	var errOut bytes.Buffer
	if rc := Run([]string{"Root"}, &bytes.Buffer{}, &errOut); rc != 1 || !strings.Contains(errOut.String(), "Use: c3po pass <vault> <search>") {
		t.Fatalf("rc = %d stderr = %q", rc, errOut.String())
	}
}

func TestMissingTokenStopsBeforeOp(t *testing.T) {
	isolate(t)
	var errOut bytes.Buffer
	if rc := Run([]string{"Root", "github"}, &bytes.Buffer{}, &errOut); rc != 1 || errOut.String() != "1Password service account token is missing.\n" {
		t.Fatalf("rc = %d stderr = %q", rc, errOut.String())
	}
}
