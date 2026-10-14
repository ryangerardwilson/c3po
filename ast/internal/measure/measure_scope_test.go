package measure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProseIsMeasuredApart(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.go"), "package a\n// b\n")
	writeFile(t, filepath.Join(root, "docs", "guide.md"), strings.Repeat("text\n", 10))
	cfg := config.Default()
	res, err := Count(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.LOC != 2 || res.ProseLOC != 10 || res.FileCount != 2 {
		t.Fatalf("loc %d prose %d files %d", res.LOC, res.ProseLOC, res.FileCount)
	}
	cfg.Enforcement.CountProse = true
	res, err = Count(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.LOC != 12 || res.ProseLOC != 10 {
		t.Fatalf("count_prose loc %d prose %d", res.LOC, res.ProseLOC)
	}
}

func TestScopeSkipsLinkedWorktreeAndSelfIgnoredDirs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.go"), "package a\n")
	writeFile(t, filepath.Join(root, "wt", ".git"), "gitdir: /elsewhere/.git/worktrees/wt\n")
	writeFile(t, filepath.Join(root, "wt", "b.go"), "package b\n")
	writeFile(t, filepath.Join(root, "cache", ".gitignore"), "# made by a tool\n*\n")
	writeFile(t, filepath.Join(root, "cache", "c.json"), "{}\n")
	writeFile(t, filepath.Join(root, "kept", ".gitignore"), "*.log\n")
	writeFile(t, filepath.Join(root, "kept", "d.go"), "package d\n")
	writeFile(t, filepath.Join(root, ".gitignore"), "artifacts/\n")
	writeFile(t, filepath.Join(root, "artifacts", "e.json"), "{}\n")

	sc := NewScope(root, config.Default())
	var seen []string
	nested, err := sc.Walk(sc.Root(), -1, func(_, rel string) error {
		seen = append(seen, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(seen, ",")
	if got != ".gitignore,a.go,kept/.gitignore,kept/d.go" {
		t.Fatalf("walk saw %s", got)
	}
	if nested != 1 {
		t.Fatalf("nested %d", nested)
	}
	if why := sc.SkipDir(filepath.Join(root, "cache")); why != SkipSelfIgnore {
		t.Fatalf("cache skip %q", why)
	}
	if why, _ := sc.SkipAncestor(filepath.Join(root, "wt", "b.go")); why != SkipNestedRepo {
		t.Fatalf("worktree skip %q", why)
	}
}
