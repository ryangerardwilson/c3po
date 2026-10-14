package measure

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestRepeatedDirtyEditsInvalidateCache(t *testing.T) {
	root := t.TempDir()
	if err := exec.Command("git", "init", root).Run(); err != nil {
		t.Skip(err)
	}
	_ = exec.Command("git", "-C", root, "config", "user.email", "t@t").Run()
	_ = exec.Command("git", "-C", root, "config", "user.name", "t").Run()
	src := filepath.Join(root, "pkg", "a.go")
	mustWrite(t, src, "package pkg\n")
	_ = exec.Command("git", "-C", root, "add", "-A").Run()
	_ = exec.Command("git", "-C", root, "commit", "-m", "i").Run()

	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()

	r1, err := CountCached(root, cfg)
	if err != nil || r1.LOC != 1 {
		t.Fatalf("%+v %v", r1, err)
	}
	mustWrite(t, src, "package pkg\n//1\n")
	r2, err := CountCached(root, cfg)
	if err != nil || r2.CacheHit || r2.LOC != 2 {
		t.Fatalf("first dirty: %+v %v", r2, err)
	}
	// second edit while already dirty — porcelain path list may be unchanged
	mustWrite(t, src, "package pkg\n//1\n//2\n")
	r3, err := CountCached(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if r3.CacheHit {
		t.Fatal("expected cache miss on second dirty edit")
	}
	if r3.LOC != 3 {
		t.Fatalf("expected loc 3 got %+v", r3)
	}
}

func TestDirtyTreeInvalidatesCache(t *testing.T) {
	root := t.TempDir()
	if err := exec.Command("git", "init", root).Run(); err != nil {
		t.Skip("git init failed")
	}
	_ = exec.Command("git", "-C", root, "config", "user.email", "t@t").Run()
	_ = exec.Command("git", "-C", root, "config", "user.name", "t").Run()
	src := filepath.Join(root, "pkg", "a.go")
	mustWrite(t, src, "package pkg\n")
	_ = exec.Command("git", "-C", root, "add", "-A").Run()
	_ = exec.Command("git", "-C", root, "commit", "-m", "init").Run()
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	_, err := CountCached(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := CountCached(root, cfg)
	if err != nil || !r2.CacheHit {
		t.Fatalf("warm hit: %+v %v", r2, err)
	}
	mustWrite(t, src, "package pkg\n//1\n//2\n//3\n//4\n//5\n")
	r3, err := CountCached(root, cfg)
	if err != nil || r3.CacheHit {
		t.Fatalf("after edit: %+v %v", r3, err)
	}
}
