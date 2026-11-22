package syncer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

func stateHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
}

func TestStateDerivesParent(t *testing.T) {
	stateHome(t)
	t.Setenv("SQLX_ROOT", "")
	t.Setenv("C3PO_SQLX", "off")
	entries := map[string]StateEntry{"a/b.txt": {RelPath: "a/b.txt", Kind: "file", DriveID: "id-1", ParentRelPath: "a"}}
	if err := SaveState("1", "1", entries); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(StateFile("1", "1"))
	if err != nil || strings.Contains(string(raw), "parent_relpath") {
		t.Fatalf("stored %v %s", err, raw)
	}
	got, err := LoadState("1", "1")
	if err != nil || got["a/b.txt"].ParentRelPath != "a" {
		t.Fatalf("loaded %v %#v", err, got)
	}
}

func TestStateMigrationDropsParent(t *testing.T) {
	stateHome(t)
	t.Setenv("SQLX_ROOT", t.TempDir())
	t.Setenv("C3PO_SQLX", "")
	path := StateFile("1", "1")
	old := `{"entries":[{"relpath":"a/b.txt","kind":"file","drive_id":"id-1","parent_relpath":"a","size":1,"mtime_ns":1},{"relpath":"c.txt","kind":"file","drive_id":"id-2","parent_relpath":"","size":1,"mtime_ns":1}]}`
	if err := sqlxdb.Write("gdrive", "state", sqlxdb.Key(path), path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadState("1", "1")
	if err != nil || got["a/b.txt"].ParentRelPath != "a" || got["c.txt"].ParentRelPath != "" || len(got) != 2 {
		t.Fatalf("loaded %v %#v", err, got)
	}
	raw, err := sqlxdb.Read("gdrive", "state", sqlxdb.Key(path), path)
	if err != nil || strings.Contains(string(raw), "parent_relpath") {
		t.Fatalf("stored %v %s", err, raw)
	}
}
