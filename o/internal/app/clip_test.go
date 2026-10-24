package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/o/internal/tui"
)

func TestCopyAndCutPasteFiles(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dest := filepath.Join(root, "dest")
	other := filepath.Join(root, "other")
	for _, dir := range []string{src, dest, other} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	note := filepath.Join(src, "note.txt")
	if err := os.WriteFile(note, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := openDir(t, root, "src")
	m = selectNamed(t, m, "note.txt")
	m = press(t, m, "y")
	if m.status != "copied note.txt" {
		t.Fatalf("status = %q", m.status)
	}
	if !strings.Contains(m.promptLine(), "copy note.txt") {
		t.Fatalf("prompt = %q", m.promptLine())
	}
	m = press(t, m, "p")
	if m.status != "note.txt is already here" {
		t.Fatalf("status = %q", m.status)
	}
	m = press(t, m, "h")
	m = selectNamed(t, m, "dest")
	m = press(t, m, "l")
	m = press(t, m, "p")
	got, err := os.ReadFile(filepath.Join(dest, "note.txt"))
	if err != nil || string(got) != "hello" {
		t.Fatalf("pasted copy = %q, %v", got, err)
	}
	if _, err := os.Stat(note); err != nil {
		t.Fatal(err)
	}
	if !m.clipSet {
		t.Fatal("copy was cleared")
	}
	if m.status != "pasted note.txt" {
		t.Fatalf("status = %q", m.status)
	}

	m = press(t, m, "h")
	m = selectNamed(t, m, "src")
	m = press(t, m, "l")
	m = selectNamed(t, m, "note.txt")
	m = press(t, m, "x")
	if m.status != "cut note.txt" {
		t.Fatalf("status = %q", m.status)
	}
	m = press(t, m, "h")
	m = selectNamed(t, m, "other")
	m = press(t, m, "l")
	m = press(t, m, "p")
	if _, err := os.Stat(filepath.Join(other, "note.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(note); !os.IsNotExist(err) {
		t.Fatalf("cut source remains: %v", err)
	}
	if m.clipSet {
		t.Fatal("cut stayed staged")
	}
	if m.status != "pasted note.txt" {
		t.Fatalf("status = %q", m.status)
	}
}

func TestCopyDirectoryAndRefuseCollisions(t *testing.T) {
	root := t.TempDir()
	box := filepath.Join(root, "box")
	other := filepath.Join(root, "other")
	if err := os.Mkdir(box, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(box, "inner.txt"), []byte("in"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}

	m := openDir(t, root, "")
	m = selectNamed(t, m, "box")
	m = press(t, m, "y")
	m = selectNamed(t, m, "other")
	m = press(t, m, "l")
	m = press(t, m, "p")
	got, err := os.ReadFile(filepath.Join(other, "box", "inner.txt"))
	if err != nil || string(got) != "in" {
		t.Fatalf("copied dir = %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(box, "inner.txt")); err != nil {
		t.Fatal(err)
	}

	m = press(t, m, "p")
	if m.status != "box exists" {
		t.Fatalf("status = %q", m.status)
	}
	if _, err := os.Stat(filepath.Join(box, "inner.txt")); err != nil {
		t.Fatal(err)
	}

	m = press(t, m, "h")
	m = selectNamed(t, m, "box")
	m = press(t, m, "l")
	m = press(t, m, "p")
	if m.status != "cannot paste a directory into itself" {
		t.Fatalf("status = %q", m.status)
	}
	if _, err := os.Stat(filepath.Join(box, "box")); !os.IsNotExist(err) {
		t.Fatalf("directory pasted into itself: %v", err)
	}
}

func TestPasteWithNothingStaged(t *testing.T) {
	m := openDir(t, t.TempDir(), "")
	m = press(t, m, "p")
	if m.status != "nothing to paste" {
		t.Fatalf("status = %q", m.status)
	}
	m = press(t, m, "x")
	if m.status != "nothing to cut" {
		t.Fatalf("status = %q", m.status)
	}
	m = press(t, m, "y")
	if m.status != "nothing to copy" {
		t.Fatalf("status = %q", m.status)
	}
}

func openDir(t *testing.T, root, name string) model {
	t.Helper()
	dir := root
	if name != "" {
		dir = filepath.Join(root, name)
	}
	m := newModel(Config{StartDir: dir, Mode: ModeBrowse})
	m.animationActive = false
	return m
}

func selectNamed(t *testing.T, m model, name string) model {
	t.Helper()
	for i, entry := range m.entries {
		if entry.Name == name {
			m.selected = i
			return m
		}
	}
	t.Fatalf("missing %s in %s", name, m.currentDir)
	return m
}

func press(t *testing.T, m model, key string) model {
	t.Helper()
	msg := previewKey(key)
	if key == "p" || key == "x" || key == "y" {
		msg = tui.KeyMsg{Type: tui.KeyRunes, Runes: []rune(key)}
	}
	next, _ := m.Update(msg)
	got, ok := next.(model)
	if !ok {
		t.Fatalf("%s returned %T", key, next)
	}
	return got
}
