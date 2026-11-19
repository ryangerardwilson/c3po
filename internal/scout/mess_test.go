package scout

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMessReportsABareRepository(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	repo := filepath.Join(t.TempDir(), "repo")
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	t.Chdir(repo)
	out, code := call("c3po", "mess")
	for _, want := range []string{"clean  0\n", "keep   0\n", "other  0\n", "  git worktree add  use c3po mess tidy add\n"} {
		if code != 0 || !bytes.Contains(out, []byte(want)) {
			t.Fatalf("code %d missing %q\n%s", code, want, out)
		}
	}
}
