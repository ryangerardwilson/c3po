package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/o/internal/tui"
)

func TestVisualRangeHighlightsAndClears(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_COLOR", "")
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(src, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := openDir(t, root, "src")
	m = selectNamed(t, m, "a.txt")
	m = press(t, m, "v")
	if !m.visual || m.status != "visual 1" {
		t.Fatalf("visual %v status %q", m.visual, m.status)
	}
	m = press(t, m, "j")
	m = pressMsg(t, m, tui.KeyMsg{Type: tui.KeyDown})
	if m.status != "visual 3" || m.entries[m.selected].Name != "c.txt" {
		t.Fatalf("status %q selected %s", m.status, m.entries[m.selected].Name)
	}
	view := m.View()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if !lineHasHighlight(view, name) {
			t.Fatalf("%s not highlighted:\n%s", name, view)
		}
	}
	m = press(t, m, "k")
	if lineHasHighlight(m.View(), "c.txt") {
		t.Fatalf("c.txt stayed highlighted:\n%s", m.View())
	}
	m = press(t, m, "v")
	if m.visual || lineHasHighlight(m.View(), "a.txt") {
		t.Fatalf("visual stayed on:\n%s", m.View())
	}
	if _, err := os.Stat(filepath.Join(src, "a.txt")); err != nil {
		t.Fatal(err)
	}

	m = selectNamed(t, m, "a.txt")
	m = press(t, m, "v")
	m = press(t, m, "k")
	if m.entries[m.selected].Name != "a.txt" || m.status != "visual 1" {
		t.Fatalf("wrapped status %q selected %s", m.status, m.entries[m.selected].Name)
	}
	m = pressMsg(t, m, tui.KeyMsg{Type: tui.KeyEsc})
	if m.visual {
		t.Fatal("esc left visual on")
	}
	m = press(t, m, "v")
	m = press(t, m, "h")
	if m.visual || m.status != "" {
		t.Fatalf("parent visual %v status %q", m.visual, m.status)
	}
	if _, err := os.Stat(filepath.Join(src, "a.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestVisualDeleteCutAndYank(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dest := filepath.Join(root, "dest")
	other := filepath.Join(root, "other")
	for _, dir := range []string{src, dest, other} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(src, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	m := openDir(t, root, "src")
	m = pressMsg(t, m, tui.KeyMsg{Type: tui.KeyDelete})
	if _, err := os.Stat(filepath.Join(src, "a.txt")); err != nil {
		t.Fatal(err)
	}

	m = selectNamed(t, m, "b.txt")
	m = press(t, m, "v")
	m = press(t, m, "k")
	m = press(t, m, "y")
	if m.visual || m.status != "copied a.txt, b.txt" {
		t.Fatalf("yank visual %v status %q", m.visual, m.status)
	}
	if !strings.Contains(m.promptLine(), "copy a.txt, b.txt") {
		t.Fatalf("prompt %q", m.promptLine())
	}
	m = press(t, m, "p")
	if m.status != "a.txt is already here" {
		t.Fatalf("status %q", m.status)
	}
	m = press(t, m, "h")
	m = selectNamed(t, m, "dest")
	m = press(t, m, "l")
	m = press(t, m, "p")
	for _, name := range []string{"a.txt", "b.txt"} {
		got, err := os.ReadFile(filepath.Join(dest, name))
		if err != nil || string(got) != name {
			t.Fatalf("copy %s = %q %v", name, got, err)
		}
		if _, err := os.Stat(filepath.Join(src, name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "c.txt")); !os.IsNotExist(err) {
		t.Fatalf("c.txt was copied: %v", err)
	}
	if !m.clipSet {
		t.Fatal("copy was cleared")
	}

	m = press(t, m, "h")
	m = selectNamed(t, m, "src")
	m = press(t, m, "l")
	m = selectNamed(t, m, "a.txt")
	m = press(t, m, "v")
	m = press(t, m, "j")
	m = press(t, m, "x")
	if m.visual || m.status != "cut a.txt, b.txt" {
		t.Fatalf("cut visual %v status %q", m.visual, m.status)
	}
	m = press(t, m, "h")
	m = selectNamed(t, m, "other")
	m = press(t, m, "l")
	m = press(t, m, "p")
	for _, name := range []string{"a.txt", "b.txt"} {
		if _, err := os.Stat(filepath.Join(other, name)); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(src, name)); !os.IsNotExist(err) {
			t.Fatalf("cut source %s remains: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(src, "c.txt")); err != nil {
		t.Fatal(err)
	}
	if m.clipSet {
		t.Fatal("cut stayed staged")
	}

	m = press(t, m, "h")
	m = selectNamed(t, m, "src")
	m = press(t, m, "l")
	m = selectNamed(t, m, "c.txt")
	m = press(t, m, "v")
	m = pressMsg(t, m, tui.KeyMsg{Type: tui.KeyDelete})
	if _, err := os.Stat(filepath.Join(src, "c.txt")); !os.IsNotExist(err) {
		t.Fatalf("c.txt remains: %v", err)
	}
	if m.visual || m.status != "deleted c.txt" {
		t.Fatalf("delete visual %v status %q", m.visual, m.status)
	}
}

func TestVisualPasteRefusesAnyExistingName(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dest := filepath.Join(root, "dest")
	for _, dir := range []string{src, dest} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(src, name), []byte("src"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dest, "b.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := openDir(t, root, "src")
	m = selectNamed(t, m, "a.txt")
	m = press(t, m, "v")
	m = press(t, m, "j")
	m = press(t, m, "y")
	m = press(t, m, "h")
	m = selectNamed(t, m, "dest")
	m = press(t, m, "l")
	m = press(t, m, "p")
	if m.status != "b.txt exists" {
		t.Fatalf("status %q", m.status)
	}
	if _, err := os.Stat(filepath.Join(dest, "a.txt")); !os.IsNotExist(err) {
		t.Fatalf("a.txt was pasted: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "b.txt"))
	if err != nil || string(got) != "keep" {
		t.Fatalf("b.txt = %q %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(src, "a.txt")); err != nil {
		t.Fatal(err)
	}
	if !m.clipSet {
		t.Fatal("refused paste cleared the stage")
	}
}

func TestVisualDDAndPartialStage(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := openDir(t, root, "")
	m = selectNamed(t, m, "a.txt")
	m = press(t, m, "v")
	m = press(t, m, "j")
	m = press(t, m, "y")
	m = selectNamed(t, m, "a.txt")
	m = press(t, m, "d")
	m = press(t, m, "d")
	if _, err := os.Stat(filepath.Join(root, "a.txt")); !os.IsNotExist(err) {
		t.Fatalf("a.txt remains: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "b.txt")); err != nil {
		t.Fatal(err)
	}
	if !m.clipSet || !strings.Contains(m.promptLine(), "copy b.txt") || strings.Contains(m.promptLine(), "a.txt") {
		t.Fatalf("stage %v prompt %q", m.clipSet, m.promptLine())
	}

	m = selectNamed(t, m, "b.txt")
	m = press(t, m, "v")
	m = press(t, m, "j")
	m = press(t, m, "d")
	m = press(t, m, "d")
	for _, name := range []string{"b.txt", "c.txt"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("%s remains: %v", name, err)
		}
	}
	if m.visual || m.clipSet || m.status != "deleted b.txt, c.txt" {
		t.Fatalf("visual %v clip %v status %q", m.visual, m.clipSet, m.status)
	}
}

func TestVisualFilterAndEmpty(t *testing.T) {
	m := openDir(t, t.TempDir(), "")
	m = press(t, m, "v")
	if m.visual || m.status != "nothing to select" {
		t.Fatalf("empty visual %v status %q", m.visual, m.status)
	}

	root := t.TempDir()
	note := filepath.Join(root, "note.txt")
	if err := os.WriteFile(note, []byte("n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m = openDir(t, root, "")
	m = press(t, m, "/")
	m = press(t, m, "v")
	if m.visual || m.input != "v" {
		t.Fatalf("filter visual %v input %q", m.visual, m.input)
	}
	m = pressMsg(t, m, tui.KeyMsg{Type: tui.KeyDelete})
	if _, err := os.Stat(note); err != nil {
		t.Fatal(err)
	}
	if m.visual {
		t.Fatal("delete entered visual mode")
	}
}

func lineHasHighlight(view, name string) bool {
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, name) && strings.Contains(line, "48;5;238") {
			return true
		}
	}
	return false
}
