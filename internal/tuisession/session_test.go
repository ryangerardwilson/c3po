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
	if err := sess.Note(Body{Args: []string{"-a"}}); err != nil {
		t.Fatal(err)
	}
	id := sess.ID()
	raw, ok, err := sqlxdb.Get(appName, kindActive, id)
	if err != nil || !ok || len(raw) == 0 {
		t.Fatalf("active %v %v %s", ok, err, raw)
	}
	if err := sess.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := sqlxdb.Get(appName, kindActive, id); err != nil || ok {
		t.Fatalf("active after close ok %v err %v", ok, err)
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
	if err := sess.Note(Body{Cursor: "x"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := sess.Index([]string{"x"}); ok {
		t.Fatal("nil index")
	}
	if err := sess.Close(); err != nil {
		t.Fatal(err)
	}
}
