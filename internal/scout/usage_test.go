package scout

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/internal/usage"
)

func TestUsageReportSplitsHumanAndAgent(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	t.Setenv("C3PO_ACTOR", "human")
	old := usage.Now
	usage.Now = func() time.Time { return time.Date(2026, 11, 11, 4, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { usage.Now = old })

	if out, code := call("c3po", "version"); code != 0 {
		t.Fatalf("version %d\n%s", code, out)
	}
	if out, code := call("c3po", "help", "-f", "ast", "patch"); code != 0 {
		t.Fatalf("help %d\n%s", code, out)
	}
	out, code := call("c3po", "usage")
	if code != 0 {
		t.Fatalf("usage %d\n%s", code, out)
	}
	unused, used, ok := splitUsage(out)
	if !ok {
		t.Fatalf("sections\n%s", out)
	}
	if !bytes.Contains(unused, []byte("\n  ast patch\n")) {
		t.Fatal("help kept ast patch alive")
	}
	if !bytes.Contains(unused, []byte("\n  ast cut\n")) {
		t.Fatal("missing unused ast cut")
	}
	if !bytes.Contains(unused, []byte("\n  ast gap\n")) {
		t.Fatal("missing unused ast gap")
	}
	if bytes.Contains(used, []byte("usage")) {
		t.Fatal("the report included itself")
	}
	version := usageLine(used, "version")
	if tailOf(version) != "1 0 0 1 2026-11-11" {
		t.Fatalf("version row %q", version)
	}
	help := usageLine(used, "help")
	if tailOf(help) != "1 0 0 1 2026-11-11" {
		t.Fatalf("help row %q", help)
	}

	t.Setenv("C3PO_ACTOR", "agent")
	if out, code := call("c3po", "version"); code != 0 {
		t.Fatalf("agent version %d\n%s", code, out)
	}
	out, code = call("c3po", "usage")
	if code != 0 {
		t.Fatalf("second %d\n%s", code, out)
	}
	_, used, ok = splitUsage(out)
	if !ok {
		t.Fatalf("second sections\n%s", out)
	}
	version = usageLine(used, "version")
	if tailOf(version) != "1 1 0 2 2026-11-11" {
		t.Fatalf("split version row %q", version)
	}
	usageRow := usageLine(used, "usage")
	if tailOf(usageRow) != "1 0 0 1 2026-11-11" {
		t.Fatalf("usage row %q", usageRow)
	}
}

func TestUsagePrefixAndExtraArgs(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	t.Setenv("C3PO_ACTOR", "service")
	out, code := call("c3po", "usage", "ast")
	if code != 0 || !bytes.Contains(out, []byte("\n  ast cut\n")) || bytes.Contains(out, []byte("\n  gmail")) {
		t.Fatalf("prefix %d\n%s", code, out)
	}
	out, code = call("c3po", "usage", "ast", "patch")
	if code != 2 || !bytes.Contains(out, []byte("usage: c3po usage [prefix]")) {
		t.Fatalf("extra %d\n%s", code, out)
	}
}

func TestUsageStoreFailureDoesNotChangeExit(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	t.Setenv("SQLX_SQLITE", "/bin/false")
	t.Setenv("C3PO_ACTOR", "human")
	out, code := call("c3po", "version")
	if code != 0 || bytes.Contains(out, []byte("sqlite")) || bytes.Contains(out, []byte("usage")) {
		t.Fatalf("version %d\n%s", code, out)
	}
}

func TestHelpNamesUsage(t *testing.T) {
	out, code := call("c3po", "help")
	if code != 0 || !bytes.Contains(out, []byte("c3po usage")) {
		t.Fatalf("help %d\n%s", code, out)
	}
}

func splitUsage(out []byte) (unused, used []byte, ok bool) {
	text := string(out)
	usedAt := strings.Index(text, "\nused\n")
	if !strings.HasPrefix(text, "unused\n") || usedAt < 0 {
		return nil, nil, false
	}
	return []byte(text[:usedAt]), []byte(text[usedAt:]), true
}

func usageLine(section []byte, name string) string {
	for _, line := range strings.Split(string(section), "\n") {
		if strings.Contains(line, name) {
			return line
		}
	}
	return ""
}

func tailOf(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return line
	}
	return strings.Join(fields[len(fields)-5:], " ")
}
