package idealake

import (
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

func TestBodyDoesNotRepeatKey(t *testing.T) {
	isolate(t)
	if _, _, code := call(t, "add", "xyz", "a quiet ending"); code != 0 {
		t.Fatal(code)
	}
	idea, ok, err := sqlxdb.Get(recordApp, recordKind, "1")
	if err != nil || !ok || strings.Contains(string(idea), `"id"`) {
		t.Fatalf("idea ok=%v err=%v %s", ok, err, idea)
	}
	if _, _, code := call(t, "add", "abc"); code != 0 {
		t.Fatal(code)
	}
	topic, ok, err := sqlxdb.Get(recordApp, recordKindTopic, "abc")
	if err != nil || !ok || strings.Contains(string(topic), `"topic"`) {
		t.Fatalf("topic ok=%v err=%v %s", ok, err, topic)
	}
}

func TestKeyMigrationDropsOldCopies(t *testing.T) {
	isolate(t)
	oldIdea := `{"id":"7","topic":"xyz","text":"an old idea","created":"2026-01-01T00:00:00Z"}`
	oldTopic := `{"topic":"xyz","created":"2026-01-01T00:00:00Z"}`
	if err := sqlxdb.Put(recordApp, recordKind, "7", []byte(oldIdea)); err != nil {
		t.Fatal(err)
	}
	if err := sqlxdb.Put(recordApp, recordKindTopic, "xyz", []byte(oldTopic)); err != nil {
		t.Fatal(err)
	}
	out, _, code := call(t, "show", "7")
	if code != 0 || !strings.Contains(out, "an old idea") {
		t.Fatalf("show %d %s", code, out)
	}
	if out, _, code := call(t, "topics"); code != 0 || !strings.Contains(out, "xyz") {
		t.Fatalf("topics %d %s", code, out)
	}
	idea, _, err := sqlxdb.Get(recordApp, recordKind, "7")
	if err != nil || strings.Contains(string(idea), `"id"`) || !strings.Contains(string(idea), `"topic":"xyz"`) {
		t.Fatalf("idea %v %s", err, idea)
	}
	topic, _, err := sqlxdb.Get(recordApp, recordKindTopic, "xyz")
	if err != nil || strings.Contains(string(topic), `"topic"`) {
		t.Fatalf("topic %v %s", err, topic)
	}
}
