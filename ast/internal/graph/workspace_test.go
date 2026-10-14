package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/packages"
)

func TestPythonDepsDoNotTouchGoModule(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("pyproject.toml", "[tool.uv.workspace]\nmembers = [\"packages/*\", \"apps/*\"]\n")
	write("packages/promise-engine/pyproject.toml", "[project]\nname = \"promise-engine\"\ndependencies = [\"model\"]\n")
	write("packages/promise-engine/engine.py", "x = 1\n")
	write("packages/model/pyproject.toml", "[project]\nname = \"model\"\n")
	write("apps/web/pyproject.toml", "[project]\nname = \"web\"\ndependencies = [\"promise-engine\"]\n")
	write("tools/metrics/go.mod", "module metrics\n")
	write("tools/metrics/main.go", "package main\nfunc main() {}\n")
	pkgs, err := packages.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	g, err := Build(root, pkgs)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Nodes) == 0 {
		t.Fatal("empty graph")
	}
	web := ID("package", "apps/web")
	engine := ID("package", "packages/promise-engine")
	model := ID("package", "packages/model")
	mod := ID("module", "tools/metrics")
	if _, ok := g.Nodes[mod]; !ok {
		t.Fatalf("missing go module node %#v", g.Nodes)
	}
	has := func(from, to string) bool {
		for _, e := range g.Out[from] {
			if e.To == to && e.Rel == "depends" {
				return true
			}
		}
		return false
	}
	if !has(web, engine) || !has(engine, model) {
		t.Fatalf("python edges %#v", g.Out)
	}
	if has(web, model) || has(mod, engine) || has(engine, mod) {
		t.Fatal("fabricated cross-language edge")
	}
}
