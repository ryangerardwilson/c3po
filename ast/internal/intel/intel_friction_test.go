package intel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func put(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSearchPagesRegexAndCase(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	for i := 0; i < 70; i++ {
		fmt.Fprintf(&b, "// Needle %d\n", i)
	}
	put(t, filepath.Join(root, "a.go"), b.String())
	cfg := config.Default()

	first, err := SearchPage(root, "Needle", cfg, SearchOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Hits) != 50 || first.Total != 70 || !first.More() {
		t.Fatalf("first page %d of %d", len(first.Hits), first.Total)
	}
	next, err := SearchPage(root, "Needle", cfg, SearchOpts{Offset: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Hits) != 20 || next.Hits[0].Line != 51 || next.More() {
		t.Fatalf("second page %d first line %d", len(next.Hits), next.Hits[0].Line)
	}
	re, err := SearchPage(root, `Needle 6\d$`, cfg, SearchOpts{Regex: true})
	if err != nil {
		t.Fatal(err)
	}
	if re.Total != 10 {
		t.Fatalf("regex total %d", re.Total)
	}
	ci, err := SearchPage(root, "needle 1", cfg, SearchOpts{IgnoreCase: true})
	if err != nil {
		t.Fatal(err)
	}
	if ci.Total != 11 {
		t.Fatalf("ignore-case total %d", ci.Total)
	}
	if _, err := SearchPage(root, "(", cfg, SearchOpts{Regex: true}); err == nil {
		t.Fatal("bad regex should fail")
	}
}

func TestWalksAgreeWithMeasureScope(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, ".gitignore"), "tests/.artifacts/\n")
	put(t, filepath.Join(root, "a.py"), "def keep():\n    return 1\n")
	put(t, filepath.Join(root, "tests", ".artifacts", "x.json"), "{\"store\": 1}\n")
	put(t, filepath.Join(root, "wt", ".git"), "gitdir: /elsewhere\n")
	put(t, filepath.Join(root, "wt", "a.py"), "def keep():\n    return 2\n")
	put(t, filepath.Join(root, ".ruff_cache", ".gitignore"), "*\n")
	cfg := config.Default()

	res, err := MapQuery(root, "", cfg, MapOptions{Depth: -1})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range res.Entries {
		if strings.HasPrefix(e.Path, "wt/") || strings.Contains(e.Path, ".artifacts") || strings.Contains(e.Path, ".ruff_cache") {
			t.Fatalf("map saw %s", e.Path)
		}
	}
	if body, err := Show(root, "keep", cfg); err != nil || !strings.Contains(body, "return 1") {
		t.Fatalf("show keep %q %v", body, err)
	}
	if _, err := Show(root, "store", cfg); err == nil {
		t.Fatal("ignored artifact key should not be a symbol")
	}
	out, err := Explain(root, ".", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "visible_files: 2") {
		t.Fatalf("explain root:\n%s", out)
	}
	out, err = Explain(root, "wt/a.py", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "skip_reason: nested git repository") || !strings.Contains(out, "search_visible: false") {
		t.Fatalf("explain nested:\n%s", out)
	}
}
