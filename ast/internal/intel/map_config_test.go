package intel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestMapMarkdownJSON(t *testing.T) {
	root := t.TempDir()
	md := filepath.Join(root, "s.md")
	if err := os.WriteFile(md, []byte("# Title\n\n### Requirement: Foo\n\n#### Scenario: Bar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	js := filepath.Join(root, "c.json")
	if err := os.WriteFile(js, []byte(`{"a":{"b":1},"c":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	ents, err := Map(root, "s.md", cfg, 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range ents {
		if e.Kind == "requirement" && strings.Contains(e.Name, "Foo") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing requirement: %+v", ents)
	}
	ents, err = Map(root, "c.json", cfg, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) == 0 {
		t.Fatal("expected json keys")
	}
}

func TestMapTOMLYAMLQMLAndDirDiscovery(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "c.toml"), []byte("[enforcement]\nmode = \"auto\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "c.yaml"), []byte("name: demo\nvalue: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ui.qml"), []byte("Item {\n  property int x: 1\n  function foo() {}\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	// directory map should discover qml via include
	ents, err := Map(root, "", cfg, 200)
	if err != nil {
		t.Fatal(err)
	}
	var sawQML, sawTOML, sawYAML bool
	for _, e := range ents {
		if strings.HasSuffix(e.Path, ".qml") {
			sawQML = true
		}
		if strings.Contains(e.Label, "section enforcement") || (e.Kind == "section" && e.Name == "enforcement") {
			sawTOML = true
		}
		if e.Kind == "key" && e.Name == "name" {
			sawYAML = true
		}
	}
	if !sawQML {
		t.Fatalf("qml not discovered in dir map: %+v", ents)
	}
	// direct maps
	te, _ := Map(root, "c.toml", cfg, 50)
	for _, e := range te {
		if e.Kind == "section" {
			sawTOML = true
		}
	}
	ye, _ := Map(root, "c.yaml", cfg, 50)
	for _, e := range ye {
		if e.Kind == "key" {
			sawYAML = true
		}
	}
	if !sawTOML || !sawYAML {
		t.Fatalf("toml=%v yaml=%v ents=%+v te=%+v ye=%+v", sawTOML, sawYAML, ents, te, ye)
	}
}
