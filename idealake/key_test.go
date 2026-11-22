package idealake

import (
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

func TestIdeaBelongsToItsTopic(t *testing.T) {
	isolate(t)
	if _, _, code := call(t, "add", "xyz", "a quiet ending"); code != 0 {
		t.Fatal(code)
	}
	if _, _, code := call(t, "add", "abc"); code != 0 {
		t.Fatal(code)
	}
	var rows []struct {
		ID    int    `json:"id"`
		Topic string `json:"topic"`
	}
	if err := sqlxdb.Query("SELECT i.id, t.name AS topic FROM idealake_idea i JOIN idealake_topic t ON t.id = i.topic_id", &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != 1 || rows[0].Topic != "xyz" {
		t.Fatalf("ideas %+v", rows)
	}
	names, err := testDeps().store.TopicNames()
	if err != nil || strings.Join(names, ",") != "abc,xyz" {
		t.Fatalf("topics %v %v", names, err)
	}
	if err := sqlxdb.Exec("DELETE FROM idealake_topic WHERE name = 'xyz'"); err == nil {
		t.Fatal("a topic with ideas was deleted")
	}
	if _, _, code := call(t, "remove", "1"); code != 0 {
		t.Fatal(code)
	}
	if _, _, code := call(t, "add", "xyz", "another"); code != 0 {
		t.Fatal(code)
	}
	ideas, err := testDeps().store.Ideas()
	if err != nil || len(ideas) != 1 || ideas[0].ID != "2" {
		t.Fatalf("a removed id was issued again: %+v %v", ideas, err)
	}
}
