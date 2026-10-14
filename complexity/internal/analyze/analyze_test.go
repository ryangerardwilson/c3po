package analyze

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/complexity/internal/config"
)

func TestRatchetIgnoresLegacy(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "t@example.com")
	runGit(t, dir, "config", "user.name", "t")
	body := "package p\nfunc hot(a, b, c, d, e, f, g int) int { return a+b+c+d+e+f+g }\n"
	if err := os.WriteFile(filepath.Join(dir, "hot.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "hot.go")
	runGit(t, dir, "commit", "-m", "init")
	touched := body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "hot.go"), []byte(touched), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(Options{Paths: []string{dir}, Config: config.Default(), Diff: "HEAD", Ratchet: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Violations {
		t.Fatalf("legacy %#v", res.Hotspots)
	}
	fresh := "package q\nfunc fresh(a bool) {\n\tif a {\n\t} else if a {\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "fresh.go"), []byte(fresh), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err = Run(Options{
		Paths: []string{dir}, Config: low(), Diff: "HEAD", Ratchet: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Violations {
		t.Fatal("expected a new violation")
	}
}

func low() config.Config {
	cfg := config.Default()
	cfg.Thresholds.Cognitive = 1
	return cfg
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %s", err, out)
	}
}
