package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/o/internal/tui"
)

func TestDDDeletesFileAndDirectory(t *testing.T) {
	root := t.TempDir()
	keep := filepath.Join(root, "keep.txt")
	note := filepath.Join(root, "note.txt")
	box := filepath.Join(root, "box")
	if err := os.WriteFile(keep, []byte("k"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(note, []byte("n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(box, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(box, "inner.txt"), []byte("in"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := openDir(t, root, "")
	m = selectNamed(t, m, "note.txt")
	m = press(t, m, "y")
	m = press(t, m, "d")
	if m.status != "d" || !m.deleteArmed {
		t.Fatalf("arm status %q armed %v", m.status, m.deleteArmed)
	}
	if _, err := os.Stat(note); err != nil {
		t.Fatal(err)
	}
	m = press(t, m, "d")
	if _, err := os.Stat(note); !os.IsNotExist(err) {
		t.Fatalf("file remains: %v", err)
	}
	if m.status != "deleted note.txt" {
		t.Fatalf("status = %q", m.status)
	}
	if m.clipSet {
		t.Fatal("deleted path stayed staged")
	}
	if m.deleteArmed {
		t.Fatal("arm stayed set")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal(err)
	}

	m = selectNamed(t, m, "box")
	m = press(t, m, "d")
	m = press(t, m, "d")
	if _, err := os.Stat(box); !os.IsNotExist(err) {
		t.Fatalf("directory remains: %v", err)
	}
	if m.status != "deleted box" {
		t.Fatalf("status = %q", m.status)
	}
}

func TestDDArmCancels(t *testing.T) {
	root := t.TempDir()
	note := filepath.Join(root, "a.txt")
	other := filepath.Join(root, "b.txt")
	if err := os.WriteFile(note, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := openDir(t, root, "")
	m = selectNamed(t, m, "a.txt")
	m = press(t, m, "d")
	m = press(t, m, "j")
	if _, err := os.Stat(note); err != nil {
		t.Fatal(err)
	}
	if m.deleteArmed {
		t.Fatal("arm stayed set")
	}
	if m.entries[m.selected].Name != "b.txt" {
		t.Fatalf("selected %s", m.entries[m.selected].Name)
	}

	m = selectNamed(t, m, "a.txt")
	m = press(t, m, "d")
	m = pressMsg(t, m, tui.KeyMsg{Type: tui.KeyEsc})
	if _, err := os.Stat(note); err != nil {
		t.Fatal(err)
	}
	if m.deleteArmed || m.status != "" {
		t.Fatalf("armed %v status %q", m.deleteArmed, m.status)
	}
}

func TestDDEmptyAndFilter(t *testing.T) {
	m := openDir(t, t.TempDir(), "")
	m = press(t, m, "d")
	m = press(t, m, "d")
	if m.status != "nothing to delete" {
		t.Fatalf("status = %q", m.status)
	}

	root := t.TempDir()
	note := filepath.Join(root, "note.txt")
	if err := os.WriteFile(note, []byte("n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m = openDir(t, root, "")
	m = press(t, m, "/")
	m = press(t, m, "d")
	if m.deleteArmed {
		t.Fatal("filter armed delete")
	}
	if _, err := os.Stat(note); err != nil {
		t.Fatal(err)
	}
	if m.input != "d" {
		t.Fatalf("filter input = %q", m.input)
	}
}

func TestDDRemovesSymlinkOnly(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	link := filepath.Join(root, "link.txt")
	if err := os.WriteFile(target, []byte("t"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	m := openDir(t, root, "")
	m = selectNamed(t, m, "link.txt")
	m = press(t, m, "d")
	m = press(t, m, "d")
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("symlink remains: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "t" {
		t.Fatalf("target = %q %v", got, err)
	}
}

func pressMsg(t *testing.T, m model, msg tui.KeyMsg) model {
	t.Helper()
	next, _ := m.Update(msg)
	got, ok := next.(model)
	if !ok {
		t.Fatalf("returned %T", next)
	}
	return got
}
