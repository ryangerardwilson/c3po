package intel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestMapShowSearchGo(t *testing.T) {
	root := t.TempDir()
	src := `package demo

func Hello() string {
	return "hi"
}

func World() {}
`
	path := filepath.Join(root, "demo.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	entries, err := Map(root, "demo.go", cfg, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 2 {
		t.Fatalf("entries %#v", entries)
	}
	body, err := Show(root, "Hello", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "return \"hi\"") {
		t.Fatalf("body %q", body)
	}
	hits, _, err := Search(root, "return", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("no hits")
	}
}
