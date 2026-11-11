package measure

import (
	"bytes"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func initRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := exec.Command("git", "init", root).Run(); err != nil {
		t.Skip(err)
	}
	_ = exec.Command("git", "-C", root, "config", "user.email", "t@t").Run()
	_ = exec.Command("git", "-C", root, "config", "user.name", "t").Run()
	return root
}

func TestSameSizeDirtyEditInvalidatesCache(t *testing.T) {
	root := initRepo(t)
	src := filepath.Join(root, "pkg", "a.go")
	mustWrite(t, src, "package pkg\n")
	_ = exec.Command("git", "-C", root, "add", "-A").Run()
	_ = exec.Command("git", "-C", root, "commit", "-m", "i").Run()

	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	mustWrite(t, src, "package pkg\n//1\n")
	before := runGit(root, "status", "--porcelain=v1", "-uall")
	if r, err := CountCached(root, cfg); err != nil || r.CacheHit {
		t.Fatalf("first dirty: %+v %v", r, err)
	}
	if r, err := CountCached(root, cfg); err != nil || !r.CacheHit {
		t.Fatalf("warm hit: %+v %v", r, err)
	}
	// same size, same porcelain path list, different bytes
	mustWrite(t, src, "package pkg\n//2\n")
	if after := runGit(root, "status", "--porcelain=v1", "-uall"); !bytes.Equal(before, after) {
		t.Fatalf("porcelain changed: %q -> %q", before, after)
	}
	r, err := CountCached(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if r.CacheHit {
		t.Fatal("expected cache miss on same-size content change")
	}
}

func TestDirtyFingerprintDigest(t *testing.T) {
	root := initRepo(t)
	body := []byte("package pkg\n")
	mustWrite(t, filepath.Join(root, "a.go"), string(body))
	out := runGit(root, "status", "--porcelain=v1", "-uall")
	sum := sha256.Sum256(body)
	h := sha256.New()
	_, _ = h.Write([]byte("porcelain\n"))
	_, _ = h.Write(out)
	_, _ = h.Write([]byte("a.go"))
	_, _ = h.Write(sum[:])
	if got := gitDirtyFingerprint(root); !bytes.Equal(got, h.Sum(nil)) {
		t.Fatalf("fingerprint %x, want %x", got, h.Sum(nil))
	}
}

func TestDirtyFingerprintDirectoryIsMissing(t *testing.T) {
	root := initRepo(t)
	nested := filepath.Join(root, "nested")
	if err := exec.Command("git", "init", nested).Run(); err != nil {
		t.Skip(err)
	}
	out := runGit(root, "status", "--porcelain=v1", "-uall")
	paths := ParsePorcelainPaths(string(out))
	if _, ok := paths["nested/"]; !ok || len(paths) != 1 {
		t.Fatalf("porcelain %q", out)
	}
	h := sha256.New()
	_, _ = h.Write([]byte("porcelain\n"))
	_, _ = h.Write(out)
	_, _ = h.Write([]byte("missing:nested/"))
	if got := gitDirtyFingerprint(root); !bytes.Equal(got, h.Sum(nil)) {
		t.Fatalf("directory should hash as missing: %x", got)
	}
}

func TestDirtyFingerprintStreamsLargeFile(t *testing.T) {
	root := initRepo(t)
	const size = 16 << 20
	if err := os.WriteFile(filepath.Join(root, "big.bin"), bytes.Repeat([]byte("x"), size), 0o644); err != nil {
		t.Fatal(err)
	}
	gitDirtyFingerprint(root) // warm git and the runtime
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	gitDirtyFingerprint(root)
	runtime.ReadMemStats(&after)
	if grew := after.TotalAlloc - before.TotalAlloc; grew > size/4 {
		t.Fatalf("fingerprint allocated %d bytes for a %d byte file; it should stream", grew, size)
	}
}
