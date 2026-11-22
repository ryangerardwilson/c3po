package syncstate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/gdrive/internal/syncer"
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
	entries := map[string]syncer.StateEntry{"a/b.txt": {RelPath: "a/b.txt", Kind: "file", DriveID: "id-1", ParentRelPath: "a"}}
	if err := (Store{}).Save("1", "1", entries); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(StateFile("1", "1"))
	if err != nil || strings.Contains(string(raw), "parent_relpath") {
		t.Fatalf("stored %v %s", err, raw)
	}
	got, err := (Store{}).Load("1", "1")
	if err != nil || got["a/b.txt"].ParentRelPath != "a" {
		t.Fatalf("loaded %v %#v", err, got)
	}
}

func TestStateRowsRoundTrip(t *testing.T) {
	stateHome(t)
	t.Setenv("SQLX_ROOT", t.TempDir())
	t.Setenv("C3PO_SQLX", "")
	entries := map[string]syncer.StateEntry{
		"a":     {RelPath: "a", Kind: "dir", DriveID: "d-1"},
		"a/b":   {RelPath: "a/b", Kind: "file", DriveID: "f-1", Size: 3, MTimeNS: 7, SHA1: "x"},
		"c.txt": {RelPath: "c.txt", Kind: "file", DriveID: "f-2", Size: 1},
	}
	if err := (Store{}).Save("1", "4", entries); err != nil {
		t.Fatal(err)
	}
	if err := (Store{}).Save("2", "4", map[string]syncer.StateEntry{"z": {RelPath: "z", Kind: "file", DriveID: "f-9"}}); err != nil {
		t.Fatal(err)
	}
	ids := func() map[string]int {
		var rows []struct {
			ID      int    `json:"id"`
			RelPath string `json:"relpath"`
		}
		if err := sqlxdb.Query("SELECT id, relpath FROM gdrive_sync_entry WHERE preset = '1'", &rows); err != nil {
			t.Fatal(err)
		}
		out := map[string]int{}
		for _, row := range rows {
			out[row.RelPath] = row.ID
		}
		return out
	}
	before := ids()
	delete(entries, "c.txt")
	entries["a/b"] = syncer.StateEntry{RelPath: "a/b", Kind: "file", DriveID: "f-1", Size: 4, MTimeNS: 8, SHA1: "y"}
	if err := (Store{}).Save("1", "4", entries); err != nil {
		t.Fatal(err)
	}
	after := ids()
	if len(after) != 2 || after["a/b"] != before["a/b"] || after["a"] != before["a"] {
		t.Fatalf("ids %v -> %v", before, after)
	}
	got, err := (Store{}).Load("1", "4")
	if err != nil || len(got) != 2 || got["a/b"].Size != 4 || got["a/b"].SHA1 != "y" || got["a/b"].ParentRelPath != "a" {
		t.Fatalf("loaded %v %#v", err, got)
	}
	if err := Delete("1", "4"); err != nil {
		t.Fatal(err)
	}
	if got, _ := (Store{}).Load("1", "4"); len(got) != 0 {
		t.Fatalf("after delete %#v", got)
	}
	if got, _ := (Store{}).Load("2", "4"); len(got) != 1 {
		t.Fatalf("other preset %#v", got)
	}
}
