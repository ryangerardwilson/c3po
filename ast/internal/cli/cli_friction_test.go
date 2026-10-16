package cli

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/files"
	"github.com/ryangerardwilson/c3po/ast/internal/trace"
)

func frictionRunner(t *testing.T, threshold int) (Runner, *bytes.Buffer, *bytes.Buffer, string) {
	t.Helper()
	ws := t.TempDir()
	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	cfg := config.Default()
	cfg.ThresholdLOC = threshold
	cfg.Cache.Dir = t.TempDir()
	if err := config.Write(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	var out, errB bytes.Buffer
	r := NewRunner(&out, &errB)
	r.Cwd = ws
	r.ConfigPath = cfgPath
	return r, &out, &errB, ws
}

func TestVersionFlagAndShowPath(t *testing.T) {
	r, out, errB, ws := frictionRunner(t, 100000)
	mustWrite(t, filepath.Join(ws, "AGENTS.md"), "# Title\n\n## Section\n")
	if code := r.Run([]string{"--version"}); code != 0 || !strings.Contains(out.String(), "0.2.0") {
		t.Fatalf("--version %d %q", code, out.String())
	}
	out.Reset()
	if code := r.Run([]string{"show", "AGENTS.md"}); code != 0 || !strings.Contains(out.String(), "h2 Section") {
		t.Fatalf("show path %d %q %q", code, out.String(), errB.String())
	}
}

func TestReadAndPatchAcrossRoots(t *testing.T) {
	r, out, errB, _ := frictionRunner(t, 1)
	other := t.TempDir()
	if err := exec.Command("git", "-C", other, "init", "-q").Run(); err != nil {
		t.Skip("git unavailable")
	}
	target := filepath.Join(other, "f.go")
	mustWrite(t, target, "package f\n// a\n// a\n")
	if code := r.Run([]string{"read", target, "--lines", "1-2"}); code != 0 || !strings.Contains(out.String(), "of 3") {
		t.Fatalf("read across roots %d %q %q", code, out.String(), errB.String())
	}
	errB.Reset()
	if code := r.Run([]string{"read", target}); code != 1 || !strings.Contains(errB.String(), "--lines 1-3") {
		t.Fatalf("whole read %d %q", code, errB.String())
	}
	sum, _ := files.HashFile(target)
	errB.Reset()
	if code := r.Run([]string{"patch", target, "--expect-hash", sum, "--old", "// a", "--new", "// b"}); code != 1 || !strings.Contains(errB.String(), "lines 2, 3") {
		t.Fatalf("ambiguous patch %d %q", code, errB.String())
	}
	out.Reset()
	if code := r.Run([]string{"patch", target, "--expect-hash", sum, "--old", "// a", "--new", "// b", "--all"}); code != 0 {
		t.Fatalf("patch --all %d %q", code, errB.String())
	}
}

func TestMoveAndSearchPaging(t *testing.T) {
	r, out, errB, ws := frictionRunner(t, 1)
	mustWrite(t, filepath.Join(ws, "a.go"), "package a\n"+strings.Repeat("// hit\n", 60))
	sum, _ := files.HashFile(filepath.Join(ws, "a.go"))
	if code := r.Run([]string{"move", "a.go", "sub/b.go", "--expect-hash", sum, "--parents"}); code != 0 || !strings.Contains(out.String(), "a.go -> sub/b.go") {
		t.Fatalf("move %d %q %q", code, out.String(), errB.String())
	}
	out.Reset()
	errB.Reset()
	if code := r.Run([]string{"search", "hit"}); code != 0 || !strings.Contains(errB.String(), "hits 1-50 of 60; next: --offset 50") {
		t.Fatalf("search page %d %q", code, errB.String())
	}
	out.Reset()
	errB.Reset()
	if code := r.Run([]string{"search", "--offset", "50", "-i", "HIT"}); code != 0 || strings.Count(out.String(), "\n") != 10 {
		t.Fatalf("search next page %d %q %q", code, out.String(), errB.String())
	}
}

func TestAmbiguousTraceJSON(t *testing.T) {
	r, out, _, ws := frictionRunner(t, 100000)
	mustWrite(t, filepath.Join(ws, "a.py"), "def f():\n    return 1\n")
	mustWrite(t, filepath.Join(ws, "b.py"), "def f():\n    return 2\n")
	if code := r.Run([]string{"impact", "--json", "f"}); code != 1 {
		t.Fatalf("ambiguous impact exit %d", code)
	}
	var res trace.Result
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatal(err, out.String())
	}
	if res.Assessment == nil || res.Assessment.Conclusion != "ambiguous" || len(res.Candidates) != 2 {
		t.Fatalf("assessment %+v", res.Assessment)
	}
}
