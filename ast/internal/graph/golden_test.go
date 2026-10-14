package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/packages"
)

func TestGoldenGraphArraysMatchCounts(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("pyproject.toml", "[tool.uv.workspace]\nmembers = [\"packages/*\", \"apps/*\"]\n")
	write("packages/promise-engine/pyproject.toml", "[project]\nname = \"promise-engine\"\ndependencies = [\"model\"]\n")
	write("packages/model/pyproject.toml", "[project]\nname = \"model\"\n")
	write("apps/web/pyproject.toml", "[project]\nname = \"web\"\ndependencies = [\"promise-engine\"]\n")
	write("tools/metrics/go.mod", "module metrics\n\ngo 1.22\n")
	write("tools/metrics/main.go", "package main\nfunc main() {}\n")
	pkgs, err := packages.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	g, err := Build(root, pkgs)
	if err != nil {
		t.Fatal(err)
	}
	nodes, edges := g.Snapshot()
	_, rels, nNodes, nEdges := g.Counts()
	if len(nodes) != nNodes || len(edges) != nEdges {
		t.Fatalf("snapshot %d/%d counts %d/%d", len(nodes), len(edges), nNodes, nEdges)
	}
	sum := 0
	for _, n := range rels {
		sum += n
	}
	if sum != len(edges) {
		t.Fatalf("relation total %d edges %d", sum, len(edges))
	}
	for _, n := range nodes {
		if g.Degree(n.ID) != len(g.Neighbors(n.ID)) {
			t.Fatalf("degree mismatch %s %d neighbors %d", n.ID, g.Degree(n.ID), len(g.Neighbors(n.ID)))
		}
		if g.RingHash(n.ID) == "" {
			t.Fatalf("empty ring %s", n.ID)
		}
	}
	for _, e := range edges {
		if (strings.Contains(e.From, "metrics") && strings.Contains(e.To, "promise")) || (strings.Contains(e.To, "metrics") && strings.Contains(e.From, "web")) {
			t.Fatalf("invented cross-language edge %+v", e)
		}
	}
	web := false
	model := false
	for _, e := range edges {
		if e.Rel == "depends" && strings.Contains(e.From, "apps/web") && strings.Contains(e.To, "promise-engine") {
			web = true
		}
		if e.Rel == "depends" && strings.Contains(e.From, "apps/web") && strings.Contains(e.To, "model") {
			t.Fatalf("web must not depend on model %+v", e)
		}
		if e.Rel == "depends" && strings.Contains(e.From, "promise-engine") && strings.Contains(e.To, "model") {
			model = true
		}
	}
	if !web || !model {
		t.Fatalf("missing python depends %#v", edges)
	}
}
