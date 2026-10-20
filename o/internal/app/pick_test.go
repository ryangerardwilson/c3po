package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/o/internal/fsnav"
)

func TestDescendPickEntersThenChooses(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(sub, "a.txt")
	if err := os.WriteFile(file, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModel(Config{StartDir: root, Mode: ModePick, Allowed: "any", Descend: true})
	idx := indexNamed(m.entries, "sub")
	if idx < 0 {
		t.Fatal(m.entries)
	}
	subPath := m.entries[idx].Path
	m.selected = idx
	next, cmd := m.activate()
	if cmd != nil || next.currentDir != subPath {
		t.Fatalf("cmd %v dir %s", cmd, next.currentDir)
	}
	fileIdx := indexNamed(next.entries, "a.txt")
	if fileIdx < 0 {
		t.Fatal(next.entries)
	}
	filePath := next.entries[fileIdx].Path
	next.selected = fileIdx
	chosen, cmd := next.accept()
	if cmd == nil || len(chosen.result.Selections) != 1 || chosen.result.Selections[0] != filePath {
		t.Fatalf("%v %#v", cmd, chosen.result.Selections)
	}
	m.selected = idx
	chosen, cmd = m.accept()
	if cmd == nil || len(chosen.result.Selections) != 1 || chosen.result.Selections[0] != subPath {
		t.Fatalf("%v %#v", cmd, chosen.result.Selections)
	}
}

func indexNamed(entries []fsnav.Entry, name string) int {
	for i, entry := range entries {
		if entry.Name == name {
			return i
		}
	}
	return -1
}
