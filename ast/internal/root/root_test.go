package root

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveGitRoot(t *testing.T) {
	base := t.TempDir()
	git := filepath.Join(base, ".git")
	if err := os.Mkdir(git, 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(base, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(nested)
	if err != nil {
		t.Fatal(err)
	}
	if got != base {
		t.Fatalf("got %s want %s", got, base)
	}
}

func TestResolveNoGit(t *testing.T) {
	base := t.TempDir()
	got, err := Resolve(base)
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(base)
	if got != abs {
		t.Fatalf("got %s want %s", got, abs)
	}
}
