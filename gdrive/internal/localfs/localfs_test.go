package localfs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestScanListsFoldersAndFilesAndSkipsLinks(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "b", "c.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "a"), filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	got, err := Tree{Root: root}.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got["a"].Kind != "dir" || got["a/b"].Kind != "dir" {
		t.Fatalf("scan = %#v", got)
	}
	file := got["a/b/c.txt"]
	if file.Kind != "file" || file.Size != 5 || file.SHA1 != "" || file.MTimeNS == 0 {
		t.Fatalf("file = %#v", file)
	}
}

func TestStatHashAndMissing(t *testing.T) {
	root := t.TempDir()
	tree := Tree{Root: root}
	if err := os.WriteFile(filepath.Join(root, "x.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	entry, err := tree.Stat("x.txt")
	if err != nil || entry.Kind != "file" || entry.Size != 5 || entry.RelPath != "x.txt" {
		t.Fatalf("stat = %#v %v", entry, err)
	}
	hash, err := tree.Hash("x.txt")
	if err != nil || hash != "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d" {
		t.Fatalf("hash = %q %v", hash, err)
	}
	if _, err := tree.Stat("missing"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing = %v", err)
	}
	if rootEntry, err := tree.Stat(""); err != nil || rootEntry.Kind != "dir" {
		t.Fatalf("root = %#v %v", rootEntry, err)
	}
}

func TestMakeDirLocateAndRelPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Music")
	tree := Tree{Root: root}
	if err := tree.MakeDir("Album/Disc 1"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(root, "Album", "Disc 1")); err != nil || !info.IsDir() {
		t.Fatalf("made %v %v", info, err)
	}
	if tree.Locate("") != root || tree.Locate("Album/song.mp3") != filepath.Join(root, "Album", "song.mp3") {
		t.Fatalf("locate %q", tree.Locate("Album/song.mp3"))
	}
	if rel, ok := tree.RelPath(filepath.Join(root, "Album", "song.docx")); !ok || rel != "Album/song.docx" {
		t.Fatalf("rel %q %v", rel, ok)
	}
	if _, ok := tree.RelPath(filepath.Join(root, "..", "elsewhere")); ok {
		t.Fatal("escaped path accepted")
	}
}
