package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashAndPatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("hello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	newH, err := PatchExact(path, h, "world", "there")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "hello there\n" {
		t.Fatalf("content %q", data)
	}
	if newH == h {
		t.Fatal("hash should change")
	}
	// stale hash
	if _, err := PatchExact(path, h, "there", "x"); err == nil {
		t.Fatal("expected stale hash error")
	}
	// ambiguous
	_ = os.WriteFile(path, []byte("aa aa\n"), 0o644)
	h2, _ := HashFile(path)
	if _, err := PatchExact(path, h2, "aa", "b"); err == nil {
		t.Fatal("expected non-unique")
	}
}

func TestReadLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := ReadLines(path, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if s != "b\nc\n" {
		t.Fatalf("%q", s)
	}
}

func TestResolvePath(t *testing.T) {
	root := t.TempDir()
	if _, err := ResolvePath(root, "../escape"); err == nil {
		t.Fatal("expected escape error")
	}
}
