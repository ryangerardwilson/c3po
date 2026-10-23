package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/o/internal/fsnav"
)

func TestOpenEntryKeepsAWorkbookInTheForeground(t *testing.T) {
	path := filepath.Join(t.TempDir(), "budget.xlsx")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	next, cmd := model{}.openEntry(fsnav.Entry{Name: "budget.xlsx", Path: path})
	if cmd == nil {
		t.Fatal("workbook was not opened in the foreground")
	}
	if next.status != "" {
		t.Fatalf("status %q", next.status)
	}
}
