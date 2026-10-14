package execgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/measure"
)

func TestParsePorcelainLeadingSpace(t *testing.T) {
	text := " M sub/a.go\n?? new.txt\n"
	m := measure.ParsePorcelainPaths(text)
	if _, ok := m["sub/a.go"]; !ok {
		t.Fatalf("got %#v", m)
	}
	if _, ok := m["new.txt"]; !ok {
		t.Fatalf("got %#v", m)
	}
	if _, ok := m["ub/a.go"]; ok {
		t.Fatal("corrupted path")
	}
}

func TestExecDeltaIgnoresUnchangedPreDirty(t *testing.T) {
	root := t.TempDir()
	if err := exec.Command("git", "init", root).Run(); err != nil {
		t.Skip(err)
	}
	_ = exec.Command("git", "-C", root, "config", "user.email", "t@t").Run()
	_ = exec.Command("git", "-C", root, "config", "user.name", "t").Run()
	must(t, os.WriteFile(filepath.Join(root, "keep.go"), []byte("package k\n"), 0o644))
	_ = exec.Command("git", "-C", root, "add", "keep.go").Run()
	_ = exec.Command("git", "-C", root, "commit", "-m", "i").Run()
	// pre-dirty untouched file
	must(t, os.WriteFile(filepath.Join(root, "dirty.go"), []byte("package d\n"), 0o644))

	cfg := config.Default()
	// use true as allowlisted "generator"
	cfg.Generators.Allow = []string{"true"}
	rep, err := Run(root, cfg, []string{"true"}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range rep.Changes {
		if c.Path == "dirty.go" {
			t.Fatalf("should not attribute pre-existing dirty: %+v", rep.Changes)
		}
	}
	// create file via generator
	cfg.Generators.Allow = []string{"touch"}
	rep, err = Run(root, cfg, []string{"touch", "generated.txt"}, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range rep.Changes {
		if c.Path == "generated.txt" && c.Status == "created" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected generated.txt created: %+v", rep)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecIgnoredDeltaUsesBeforeAndAfterContent(t *testing.T) {
	root := t.TempDir()
	if err := exec.Command("git", "init", root).Run(); err != nil {
		t.Skip(err)
	}
	must(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*.log\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "unchanged.log"), []byte("same\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "modified.log"), []byte("before\n"), 0o644))

	cfg := config.Default()
	cfg.Generators.Allow = []string{"sh -c"}
	rep, err := Run(root, cfg, []string{"sh", "-c", "printf after > modified.log; printf new > created.log"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.AuditComplete {
		t.Fatalf("expected complete audit: %+v", rep.AuditNotes)
	}
	got := map[string]string{}
	for _, c := range rep.IgnoredObserved {
		got[c.Path] = c.Status
	}
	if got["modified.log"] != "modified_ignored" {
		t.Fatalf("modified ignored path missing: %+v", rep.IgnoredObserved)
	}
	if got["created.log"] != "created_ignored" {
		t.Fatalf("created ignored path missing: %+v", rep.IgnoredObserved)
	}
	if _, ok := got["unchanged.log"]; ok {
		t.Fatalf("unchanged ignored path attributed to generator: %+v", rep.IgnoredObserved)
	}
}

func TestExecDetectsDirtyPathReturningClean(t *testing.T) {
	root := t.TempDir()
	if err := exec.Command("git", "init", root).Run(); err != nil {
		t.Skip(err)
	}
	_ = exec.Command("git", "-C", root, "config", "user.email", "t@t").Run()
	_ = exec.Command("git", "-C", root, "config", "user.name", "t").Run()
	path := filepath.Join(root, "a.go")
	must(t, os.WriteFile(path, []byte("package a\n"), 0o644))
	_ = exec.Command("git", "-C", root, "add", "a.go").Run()
	_ = exec.Command("git", "-C", root, "commit", "-m", "initial").Run()
	must(t, os.WriteFile(path, []byte("package a\n// dirty\n"), 0o644))

	cfg := config.Default()
	cfg.Generators.Allow = []string{"git checkout --"}
	rep, err := Run(root, cfg, []string{"git", "checkout", "--", "a.go"}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range rep.Changes {
		if c.Path == "a.go" && c.Status == "modified" && c.Hash != "" {
			return
		}
	}
	t.Fatalf("dirty-to-clean content delta missing: %+v", rep.Changes)
}
