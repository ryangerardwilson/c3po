package trace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestAmbiguousAssessmentAndLinkedWorktree(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		must(t, os.MkdirAll(filepath.Dir(p), 0o755))
		must(t, os.WriteFile(p, []byte(body), 0o644))
	}
	write("a.py", "def target():\n    return 1\n\ndef caller():\n    return target()\n")
	write("wt/.git", "gitdir: /elsewhere/.git/worktrees/wt\n")
	write("wt/a.py", "def target():\n    return 2\n")
	cfg := config.Default()

	res, err := Query(root, cfg, "target", "callers", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ambiguous || res.Resolved.Path != "a.py" {
		t.Fatalf("linked worktree was indexed: %+v", res.Candidates)
	}

	write("b.py", "def target():\n    return 3\n")
	res, err = Query(root, cfg, "target", "impact", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ambiguous || res.Assessment == nil || res.Assessment.Conclusion != "ambiguous" {
		t.Fatalf("ambiguous assessment %+v", res.Assessment)
	}
}
