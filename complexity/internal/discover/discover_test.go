package discover

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/complexity/internal/config"
)

func TestWalkSkipsTestsVendorAndNestedGit(t *testing.T) {
	root := t.TempDir()
	must := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must("main.go", "package main\nfunc main() {}\n")
	must("main_test.go", "package main\n")
	must("web/src/a.test.ts", "export {}\n")
	must("web/src/a.ts", "export {}\n")
	must("node_modules/pkg/index.js", "module.exports = {}\n")
	must("vendor/x.go", "package vendor\n")
	must("nested/keep.go", "package nested\n")
	if err := os.MkdirAll(filepath.Join(root, "nested", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	files, err := Walk(root, config.Default(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, f := range files {
		got[f.Path] = true
	}
	if !got["main.go"] || !got["web/src/a.ts"] {
		t.Fatalf("missing sources: %#v", got)
	}
	for _, banned := range []string{"main_test.go", "web/src/a.test.ts", "node_modules/pkg/index.js", "vendor/x.go", "nested/keep.go"} {
		if got[banned] {
			t.Fatalf("analyzed %s in %#v", banned, got)
		}
	}
}

func TestIncludeTests(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "foo_test.go")
	if err := os.WriteFile(p, []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := Walk(root, config.Default(), Options{IncludeTests: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != "foo_test.go" {
		t.Fatalf("%#v", files)
	}
}
