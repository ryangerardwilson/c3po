package measure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestCountAndExclude(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "src", "a.go"), "package main\n\nfunc main() {}\n")
	mustWrite(t, filepath.Join(root, "node_modules", "x.js"), "const x = 1\nconst y = 2\n")
	mustWrite(t, filepath.Join(root, "src", "bin.dat"), "a\x00b\n")

	cfg := config.Default()
	res, err := Count(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.LOC != 3 {
		t.Fatalf("loc=%d want 3", res.LOC)
	}
	if res.FileCount != 1 {
		t.Fatalf("files=%d", res.FileCount)
	}
}

func TestPathMatches(t *testing.T) {
	cfg := config.Default()
	if !PathMatches(cfg, "src/foo.go") {
		t.Fatal("expected match")
	}
	if PathMatches(cfg, "node_modules/x.js") {
		t.Fatal("expected exclude")
	}
}

func TestGlobBrace(t *testing.T) {
	g := compileGlobs([]string{"**/*.{go,ts}"})
	if !matchAny(g, "pkg/a.go") {
		t.Fatal("go")
	}
	if !matchAny(g, "pkg/a.ts") {
		t.Fatal("ts")
	}
	if matchAny(g, "pkg/a.py") {
		t.Fatal("py should not match")
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
