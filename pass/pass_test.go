package pass

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

// isolate keeps the real nebula binary out of reach.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN", "")
	t.Setenv("OP_SERVICE_ACCOUNT_TOKEN_FILE", "")
}

func TestRunNeedsNebula(t *testing.T) {
	isolate(t)
	var errOut bytes.Buffer
	if rc := Run([]string{"Root", "github"}, &bytes.Buffer{}, &errOut); rc != 1 || !strings.Contains(errOut.String(), "nebula is not installed") {
		t.Fatalf("rc = %d stderr = %q", rc, errOut.String())
	}
}

func TestForwardsCryptArgs(t *testing.T) {
	prevLook, prevRun := lookNebula, runCommand
	t.Cleanup(func() { lookNebula, runCommand = prevLook, prevRun })
	var got []string
	lookNebula = func(string) (string, error) { return "/bin/nebula", nil }
	runCommand = func(path string, args []string, stdout, _ io.Writer) int {
		got = append([]string{path}, args...)
		fmt.Fprintln(stdout, "0.1.0")
		return 0
	}
	var out bytes.Buffer
	if rc := Run([]string{"version"}, &out, &bytes.Buffer{}); rc != 0 || out.String() != "0.1.0\n" {
		t.Fatalf("rc = %d out = %q", rc, out.String())
	}
	if strings.Join(got, " ") != "/bin/nebula crypt version" {
		t.Fatalf("%v", got)
	}
}

func TestScreenHelpDescribesCopy(t *testing.T) {
	for _, needle := range []string{"r2d2 pass", "vault", "item", "enter copies the selected item", "secret is not printed", "dd deletes the selected item", "nebula crypt"} {
		if !strings.Contains(ScreenHelp, needle) {
			t.Fatalf("screen help missing %q", needle)
		}
	}
	if strings.Contains(ScreenHelp, "c3po pass") {
		t.Fatal(ScreenHelp)
	}
}

func TestScreenNeedsNebula(t *testing.T) {
	isolate(t)
	var errOut bytes.Buffer
	if rc := Screen(&bytes.Buffer{}, &errOut); rc != 1 || !strings.Contains(errOut.String(), "nebula is not installed") {
		t.Fatalf("rc = %d stderr = %q", rc, errOut.String())
	}
}
