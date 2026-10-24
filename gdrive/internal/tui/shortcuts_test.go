package tui

import (
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
)

func TestBrowseShortcutsStayBehindQuestionMark(t *testing.T) {
	m := &model{
		entries:   []driveapi.NavEntry{{ID: "a", Name: "a.txt", MimeType: "text/plain"}},
		pathLabel: "My Drive",
		height:    24,
		width:     80,
	}
	before := m.View()
	if strings.Contains(before, "j/k move") || !strings.Contains(before, "a.txt") {
		t.Fatalf("list\n%s", before)
	}
	if m.handleKey("?") || m.cursor != 0 {
		t.Fatalf("cursor %d", m.cursor)
	}
	help := m.View()
	if !strings.Contains(help, "upload") || !strings.Contains(help, "delete") || strings.Contains(help, "a.txt") {
		t.Fatalf("help\n%s", help)
	}
	if m.handleKey("j") || m.cursor != 0 || !m.help {
		t.Fatalf("cursor %d help %v", m.cursor, m.help)
	}
	if m.handleKey("esc") || m.help || !strings.Contains(m.View(), "a.txt") || strings.Contains(m.View(), "j/k move") {
		t.Fatalf("esc\n%s", m.View())
	}
	if m.handleKey("?") || !m.handleKey("q") {
		t.Fatal("q")
	}
}
