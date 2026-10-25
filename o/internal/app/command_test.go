package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/o/internal/tui"
)

func TestViewFillsTerminalHeight(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModel(Config{StartDir: root, Mode: ModeBrowse})
	m.height = 24
	m.animationActive = false
	m.status = "ready"
	lines := strings.Split(m.View(), "\n")
	if len(lines) != 24 {
		t.Fatalf("lines = %d, want 24", len(lines))
	}
	view := strings.Join(lines, "\n")
	if !strings.Contains(lines[0], "o") || !strings.Contains(lines[len(lines)-1], "ready") || strings.Contains(view, "j/k move") {
		t.Fatalf("frame edges:\n%s\n%s", lines[0], lines[len(lines)-1])
	}
	if !strings.Contains(view, "a.txt") {
		t.Fatal("listing missing a.txt")
	}

	empty := newModel(Config{StartDir: t.TempDir(), Mode: ModeBrowse})
	empty.height = 24
	empty.animationActive = false
	emptyLines := strings.Split(empty.View(), "\n")
	emptyView := strings.Join(emptyLines, "\n")
	if len(emptyLines) != 24 || strings.Contains(emptyView, "j/k move") || strings.Contains(emptyLines[len(emptyLines)-1], "quit") {
		t.Fatalf("empty frame lines = %d, last = %q", len(emptyLines), emptyLines[len(emptyLines)-1])
	}
	if !strings.Contains(emptyView, "No entries.") {
		t.Fatal("empty listing missing the empty state")
	}

	next, _ := m.Update(previewKey("?"))
	helped, ok := next.(model)
	if !ok || !helped.showHelp {
		t.Fatalf("help type %T", next)
	}
	help := helped.View()
	if !strings.Contains(help, "cut") || !strings.Contains(help, "copy") || !strings.Contains(help, "paste") || !strings.Contains(help, "quit") {
		t.Fatalf("help\n%s", help)
	}
	selected := helped.selected
	next, _ = helped.Update(previewKey("j"))
	held, ok := next.(model)
	if !ok || !held.showHelp || held.selected != selected {
		t.Fatalf("j while help selected %d show %v", held.selected, held.showHelp)
	}
	next, _ = held.Update(previewKey("?"))
	back, ok := next.(model)
	if !ok || back.showHelp || !strings.Contains(back.View(), "a.txt") || strings.Contains(back.View(), "j/k move") {
		t.Fatalf("closed\n%s", back.View())
	}
}

func TestBangCreatesRemovesAndRefuses(t *testing.T) {
	root := t.TempDir()
	m := newModel(Config{StartDir: root, Mode: ModeBrowse})
	m.animationActive = false

	m = submitCommand(t, m, "mkdir notes")
	info, err := os.Stat(filepath.Join(root, "notes"))
	if err != nil || !info.IsDir() {
		t.Fatalf("mkdir notes: %v", err)
	}
	if m.entries[m.selected].Name != "notes" {
		t.Fatalf("selected = %q", m.entries[m.selected].Name)
	}

	m = submitCommand(t, m, "touch notes/a.txt")
	if _, err := os.Stat(filepath.Join(root, "notes", "a.txt")); err != nil {
		t.Fatal(err)
	}

	m = submitCommand(t, m, "rm -rf notes")
	if _, err := os.Stat(filepath.Join(root, "notes")); err != nil {
		t.Fatalf("refused command removed notes: %v", err)
	}
	if m.status != "flag -rf is not allowed" {
		t.Fatalf("status = %q", m.status)
	}

	m = submitCommand(t, m, "rm notes/a.txt")
	if _, err := os.Stat(filepath.Join(root, "notes", "a.txt")); !os.IsNotExist(err) {
		t.Fatalf("rm notes/a.txt: %v", err)
	}
}

func TestBangEscCancels(t *testing.T) {
	root := t.TempDir()
	m := newModel(Config{StartDir: root, Mode: ModeBrowse})
	next, _ := m.Update(previewKey("!"))
	m = next.(model)
	next, _ = m.Update(previewKey("mkdir notes"))
	m = next.(model)
	next, _ = m.Update(tui.KeyMsg{Type: tui.KeyEsc})
	m = next.(model)
	if m.commanding || m.status != "command cancelled" {
		t.Fatalf("commanding = %v status = %q", m.commanding, m.status)
	}
	if _, err := os.Stat(filepath.Join(root, "notes")); !os.IsNotExist(err) {
		t.Fatalf("cancelled command created notes: %v", err)
	}
}

func TestBangCommandLineIsTheLastRow(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModel(Config{StartDir: root, Mode: ModeBrowse})
	m.height = 24
	m.width = 80
	m.animationActive = false
	m.status = "ready"
	next, _ := m.Update(previewKey("!"))
	m = next.(model)
	lines := strings.Split(m.View(), "\n")
	if len(lines) != 24 || !strings.Contains(lines[len(lines)-1], "touch, mkdir, or rm") {
		t.Fatalf("open last = %q lines %d", lines[len(lines)-1], len(lines))
	}
	if strings.Contains(lines[1], "touch") {
		t.Fatalf("prompt = %q", lines[1])
	}
	next, _ = m.Update(previewKey("mkdir notes"))
	m = next.(model)
	lines = strings.Split(m.View(), "\n")
	if len(lines) != 24 || !strings.Contains(lines[len(lines)-1], "! mkdir notes|") {
		t.Fatalf("last = %q", lines[len(lines)-1])
	}
	if strings.Contains(lines[1], "mkdir") || strings.Contains(lines[1], "!") {
		t.Fatalf("prompt = %q", lines[1])
	}
	if !strings.Contains(strings.Join(lines, "\n"), "a.txt") {
		t.Fatal("listing hidden")
	}
}

func submitCommand(t *testing.T, m model, line string) model {
	t.Helper()
	next, _ := m.Update(previewKey("!"))
	m, ok := next.(model)
	if !ok || !m.commanding {
		t.Fatalf("! did not open the prompt: %#v", next)
	}
	next, _ = m.Update(previewKey(line))
	m, ok = next.(model)
	if !ok {
		t.Fatalf("typing returned %T", next)
	}
	next, _ = m.Update(tui.KeyMsg{Type: tui.KeyEnter})
	m, ok = next.(model)
	if !ok {
		t.Fatalf("enter returned %T", next)
	}
	if m.commanding {
		t.Fatal("prompt stayed open")
	}
	return m
}
