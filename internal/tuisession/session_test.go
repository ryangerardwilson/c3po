package tuisession

import (
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

func TestTakeResume(t *testing.T) {
	id, rest, err := TakeResume([]string{"--resume", "01a17622-5fd9-7bc2-81ff-3e200a8c16b7", "-a"})
	if err != nil || id != "01a17622-5fd9-7bc2-81ff-3e200a8c16b7" || len(rest) != 1 || rest[0] != "-a" {
		t.Fatalf("id %q rest %v err %v", id, rest, err)
	}
	_, rest, err = TakeResume([]string{"keep", "--resume=01a17622-5fd9-7bc2-81ff-3e200a8c16b7"})
	if err != nil || len(rest) != 1 || rest[0] != "keep" {
		t.Fatalf("equals rest %v err %v", rest, err)
	}
	if _, _, err := TakeResume([]string{"--resume"}); err == nil {
		t.Fatal("bare --resume")
	}
	if _, _, err := TakeResume([]string{"--resume=nope"}); err == nil {
		t.Fatal("bad id")
	}
}

func TestSessionRoundTrip(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SQLX_ROOT", root)
	sess, err := Open("looptab", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.Place("", "job-a"); err != nil {
		t.Fatal(err)
	}
	if err := sess.Note(Place{Args: []string{"-a"}}); err != nil {
		t.Fatal(err)
	}
	id := sess.ID()
	if n := activeRows(t, id); n != 1 {
		t.Fatalf("active rows %d", n)
	}
	if err := sess.Close(); err != nil {
		t.Fatal(err)
	}
	if n := activeRows(t, id); n != 0 {
		t.Fatalf("active rows after close %d", n)
	}
	again, err := Open("looptab", id)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	body, ok := again.Restored()
	if !ok || body.Cursor != "job-a" || body.Screen != "looptab" || len(body.Args) != 1 || body.Args[0] != "-a" {
		t.Fatalf("body %+v ok %v", body, ok)
	}
	if i, found := again.Index([]string{"job-b", "job-a"}); !found || i != 1 {
		t.Fatalf("index %d %v", i, found)
	}
	if _, err := Open("looptab", "01a17622-5fd9-7bc2-81ff-3e200a8c16b7"); err == nil {
		t.Fatal("missing session")
	}
	if _, err := Open("deport", id); err == nil {
		t.Fatal("wrong screen")
	}
	if _, err := filepath.Abs(filepath.Join(root, "data", "c3po.sqlite")); err != nil {
		t.Fatal(err)
	}
}

func TestNilSessionIsQuiet(t *testing.T) {
	var sess *Session
	if err := sess.Place("p", "c"); err != nil {
		t.Fatal(err)
	}
	if err := sess.Note(Place{Cursor: "x"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := sess.Index([]string{"x"}); ok {
		t.Fatal("nil index")
	}
	if err := sess.Close(); err != nil {
		t.Fatal(err)
	}
}

func activeRows(t *testing.T, id string) int {
	t.Helper()
	var rows []struct {
		N int `json:"n"`
	}
	if err := sqlxdb.Query("SELECT count(*) AS n FROM tui_active WHERE session_id = "+sqlxdb.Quote(id)+" AND pid > 0", &rows); err != nil {
		t.Fatal(err)
	}
	return rows[0].N
}

func TestArgsStayNilUntilNoted(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	sess, err := Open("l", "")
	if err != nil {
		t.Fatal(err)
	}
	sess.Close()
	again, err := Open("l", sess.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	if body, _ := again.Restored(); body.Args != nil {
		t.Fatalf("args %#v", body.Args)
	}
}

func TestAKeyDoesNotWaitAndOnlyAChangeIsStored(t *testing.T) {
	t.Setenv("SQLX_ROOT", t.TempDir())
	sess, err := Open("o", "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		if err := sess.Place("/tmp", "same"); err != nil {
			t.Fatal(err)
		}
	}
	if err := sess.Place("/tmp", "last"); err != nil {
		t.Fatal(err)
	}
	if err := sess.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := Open("o", sess.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	if body, ok := again.Restored(); !ok || body.Cursor != "last" || body.Path != "/tmp" {
		t.Fatalf("close did not store the last place: %+v", body)
	}
	sess.mu.Lock()
	written := sess.written
	sess.mu.Unlock()
	if written != placeKey(Place{Screen: "o", CWD: workingDir(""), Path: "/tmp", Cursor: "last"}) {
		t.Fatalf("written %s", written)
	}
}
