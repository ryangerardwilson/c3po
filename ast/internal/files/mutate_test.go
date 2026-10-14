package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndBatchPatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "n.go")
	h, err := CreateAbsent(path, []byte("package n\nconst A=1\nconst B=2\n"), false, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateAbsent(path, []byte("x"), false, 0o644); err == nil {
		t.Fatal("expected exists error")
	}
	h2, err := PatchOps(path, h, []Op{
		{Old: "A=1", New: "A=10"},
		{Old: "B=2", New: "B=20"},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "package n\nconst A=10\nconst B=20\n" {
		t.Fatalf("%q", data)
	}
	if h2 == h {
		t.Fatal("hash should change")
	}
}

func TestDeleteExactAndPrune(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a", "b", "f.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("delete me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pruned, err := DeleteExact(root, path, h, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(pruned) != 2 {
		t.Fatalf("pruned %v", pruned)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "a")); !os.IsNotExist(err) {
		t.Fatalf("empty ancestors remain: %v", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("workspace root removed: %v", err)
	}
}

func TestDeleteExactRejectsUnsafeTargets(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "f.txt")
	if err := os.WriteFile(path, []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := DeleteExact(root, path, "deadbeef", false); err == nil {
		t.Fatal("expected stale hash error")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stale hash removed file: %v", err)
	}
	if _, err := DeleteExact(root, root, "deadbeef", false); err == nil {
		t.Fatal("expected root rejection")
	}
	if _, err := DeleteExact(root, filepath.Join(root, "..", "escape"), "deadbeef", false); err == nil {
		t.Fatal("expected outside-root rejection")
	}
	if _, err := DeleteExact(root, root, "deadbeef", true); err == nil {
		t.Fatal("expected directory rejection")
	}

	target := filepath.Join(root, "target.txt")
	link := filepath.Join(root, "link.txt")
	if err := os.WriteFile(target, []byte("target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	h, err := HashFile(link)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeleteExact(root, link, h, false); err == nil {
		t.Fatal("expected symlink rejection")
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("symlink target removed: %v", err)
	}

	external := t.TempDir()
	externalFile := filepath.Join(external, "external.txt")
	if err := os.WriteFile(externalFile, []byte("external\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	parentLink := filepath.Join(root, "linked-dir")
	if err := os.Symlink(external, parentLink); err != nil {
		t.Fatal(err)
	}
	h, err = HashFile(filepath.Join(parentLink, "external.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeleteExact(root, filepath.Join(parentLink, "external.txt"), h, true); err == nil {
		t.Fatal("expected symlink-ancestor rejection")
	}
	if _, err := os.Stat(externalFile); err != nil {
		t.Fatalf("outside file removed: %v", err)
	}
}

func TestDeleteExactPruningStopsAtNonEmptyAncestor(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a", "b", "f.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("delete me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(root, "a", "keep.txt")
	if err := os.WriteFile(keep, []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pruned, err := DeleteExact(root, path, h, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(pruned) != 1 || pruned[0] != filepath.Join(root, "a", "b") {
		t.Fatalf("unexpected pruned directories %v", pruned)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("non-empty ancestor was removed: %v", err)
	}
}
