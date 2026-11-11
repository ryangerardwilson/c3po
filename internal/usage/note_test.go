package usage

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

func TestNoteCountsDayActorAndSuccess(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SQLX_ROOT", root)
	t.Setenv("C3PO_ACTOR", "human")
	old := Now
	Now = func() time.Time { return time.Date(2026, 11, 11, 3, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { Now = old })

	Note(0, []string{"c3po", "ast", "hash", "secret-path"})
	Note(1, []string{"c3po", "ast", "hash", "other"})
	t.Setenv("C3PO_ACTOR", "agent")
	Now = func() time.Time { return time.Date(2026, 11, 12, 3, 0, 0, 0, time.UTC) }
	Note(0, []string{"c3po", "ast", "hash"})

	rows, err := sqlxdb.ListUsage()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows %#v", rows)
	}
	var human, agent sqlxdb.UsageRow
	for _, row := range rows {
		if row.Feature != "ast hash" {
			t.Fatalf("feature %q", row.Feature)
		}
		switch row.Actor {
		case "human":
			human = row
		case "agent":
			agent = row
		default:
			t.Fatalf("actor %s", row.Actor)
		}
	}
	if human.N != 2 || human.OK != 1 || human.Day != "2026-11-11" {
		t.Fatalf("human %#v", human)
	}
	if agent.N != 1 || agent.OK != 1 || agent.Day != "2026-11-12" {
		t.Fatalf("agent %#v", agent)
	}
	body, err := os.ReadFile(filepath.Join(root, "data", "c3po.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(body, []byte("secret-path")) {
		t.Fatal("the database stored an argument")
	}
	orm, err := os.ReadFile(filepath.Join(root, "orm", "c3po"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(orm, []byte("model usage")) {
		t.Fatalf("orm\n%s", orm)
	}
}

func TestNoteSkipsWhenSQLXIsOff(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SQLX_ROOT", root)
	t.Setenv("C3PO_SQLX", "off")
	t.Setenv("C3PO_ACTOR", "human")
	Note(0, []string{"c3po", "version"})
	if _, err := os.Stat(filepath.Join(root, "data", "c3po.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("sqlite created: %v", err)
	}
}

func TestNoteIgnoresStoreFailure(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	t.Setenv("SQLX_SQLITE", "/bin/false")
	t.Setenv("C3PO_ACTOR", "human")
	Note(0, []string{"c3po", "version"})
}

func TestNoteSkipsRealDatabaseDuringTests(t *testing.T) {
	t.Setenv("SQLX_ROOT", "")
	t.Setenv("C3PO_ACTOR", "human")
	Note(0, []string{"c3po", "version"})
}
