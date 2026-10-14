package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/packages"
)

func TestImportReachAndCuts(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("Apps/a/go.mod", "module example.com/a\n\ngo 1.26\n")
	write("Apps/a/lib.go", "package a\n")
	write("Apps/a/install.sh", "#!/bin/sh\n")
	write("Apps/b/go.mod", "module example.com/b\n\ngo 1.26\n")
	write("Apps/b/b.go", "package b\nimport \"example.com/a\"\n")
	write("Apps/b/install.sh", "#!/bin/sh\n")
	write("Apps/c/go.mod", "module example.com/c\n\ngo 1.26\n")
	write("Apps/c/c.go", "package c\nimport \"example.com/b\"\n")
	write("Apps/c/install.sh", "#!/bin/sh\n")

	pkgs, err := packages.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	g, err := Build(root, pkgs)
	if err != nil {
		t.Fatal(err)
	}
	n, ok := g.Resolve("Apps/b/b.go")
	if !ok || n.Kind != "file" {
		t.Fatalf("resolve file %#v %v", n, ok)
	}
	nb := g.Neighbors(n.ID)
	hasImport := false
	for _, e := range nb {
		if e.Rel == "imports" && e.To == ID("gopkg", "Apps/a") {
			hasImport = true
		}
	}
	if !hasImport {
		t.Fatalf("missing imports edge: %#v", nb)
	}
	if g.RingHash(n.ID) == "" {
		t.Fatal("empty ring hash")
	}
	got := g.ReverseImportReach([]string{"Apps/a/lib.go"})
	join := stringsJoin(got)
	if !contains(got, "Apps/a") || !contains(got, "Apps/b") || !contains(got, "Apps/c") {
		t.Fatalf("reach %s", join)
	}
	nk, _, nn, ne := g.Counts()
	if nn == 0 || ne == 0 || nk["gopkg"] < 3 {
		t.Fatalf("counts nodes=%d edges=%d kinds=%v", nn, ne, nk)
	}
	if len(g.ImportComponents()) == 0 {
		t.Fatal("no components")
	}
	cuts := g.CutVertices()
	if !contains(cuts, ID("gopkg", "Apps/b")) {
		t.Fatalf("expected Apps/b cut, got %v", cuts)
	}
}

func stringsJoin(xs []string) string {
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += ","
		}
		out += x
	}
	return out
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
