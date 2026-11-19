package hook

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

// worktreeRepo is a git repository, enforced when enforced is true.
func worktreeRepo(t *testing.T, enforced bool) (string, config.Config) {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(repo, "a.go"), []byte("package a\n//1\n//2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfg.Enforcement.Mode = "auto"
	if enforced {
		cfg.ThresholdLOC = 1
	}
	return repo, cfg
}

func decideShell(t *testing.T, command, cwd string, cfg config.Config) Decision {
	t.Helper()
	req := Request{ToolName: "Bash", ToolInput: map[string]any{"command": command}, Cwd: cwd}
	dec, err := Evaluate(req, cfg, cwd, "")
	if err != nil {
		t.Fatal(err)
	}
	return dec
}

func TestWorktreeAddIsDeniedEnforcedOrNot(t *testing.T) {
	for _, enforced := range []bool{false, true} {
		repo, cfg := worktreeRepo(t, enforced)
		outside := t.TempDir()
		cases := []struct{ cmd, cwd string }{
			{"git worktree add ../x -b x", repo},
			{"git worktree add /tmp/elsewhere", repo},
			{"git -C " + repo + " worktree add ../x", outside},
			{"bash -c 'git worktree add ../x'", repo},
			{"sh -c \"git -C " + repo + " worktree add ../y\"", outside},
			{"cd " + repo + " && git worktree add ../x", outside},
		}
		for _, c := range cases {
			dec := decideShell(t, c.cmd, c.cwd, cfg)
			if dec.PermissionDecision != "deny" || !strings.Contains(dec.PermissionDecisionReason, "use c3po mess tidy add") {
				t.Fatalf("enforced=%v %q: %+v", enforced, c.cmd, dec)
			}
			if strings.Contains(dec.PermissionDecisionReason, "run c3po mess tidy first") {
				t.Fatalf("nothing is clean: %s", dec.PermissionDecisionReason)
			}
		}
	}
}

func TestWorktreeReadsStayAllowed(t *testing.T) {
	for _, enforced := range []bool{false, true} {
		repo, cfg := worktreeRepo(t, enforced)
		for _, cmd := range []string{
			"git worktree list",
			"git worktree remove ../x",
			"c3po mess",
			"c3po mess tidy",
			"c3po mess tidy --session s1",
			"c3po mess tidy add ../x -b x",
		} {
			if dec := decideShell(t, cmd, repo, cfg); dec.PermissionDecision != "allow" {
				t.Fatalf("enforced=%v %q: %+v", enforced, cmd, dec)
			}
		}
	}
}

func TestWorktreeAddOutsideARepositoryIsAllowed(t *testing.T) {
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	if dec := decideShell(t, "git worktree add x", t.TempDir(), cfg); dec.PermissionDecision != "allow" {
		t.Fatalf("%+v", dec)
	}
}

func TestTidyAddDeniedWhileCleanExists(t *testing.T) {
	repo, cfg := worktreeRepo(t, false)
	old := cleanWorktrees
	cleanWorktrees = func(string) []string { return []string{"/w/done"} }
	t.Cleanup(func() { cleanWorktrees = old })

	dec := decideShell(t, "c3po mess tidy add ../x", repo, cfg)
	if dec.PermissionDecision != "deny" || !strings.Contains(dec.PermissionDecisionReason, "run c3po mess tidy first  /w/done") {
		t.Fatalf("%+v", dec)
	}
	dec = decideShell(t, "git worktree add ../x", repo, cfg)
	if dec.PermissionDecision != "deny" || !strings.Contains(dec.PermissionDecisionReason, "use c3po mess tidy add; run c3po mess tidy first") {
		t.Fatalf("%+v", dec)
	}
	for _, cmd := range []string{"c3po mess", "c3po mess tidy", "git worktree list"} {
		if dec := decideShell(t, cmd, repo, cfg); dec.PermissionDecision != "allow" {
			t.Fatalf("%q: %+v", cmd, dec)
		}
	}
}
