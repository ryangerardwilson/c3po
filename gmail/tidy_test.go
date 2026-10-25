package gmail

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/gmail/internal/api"
)

func TestTidyTrashesConfiguredSpam(t *testing.T) {
	box := &fakeBox{ids: map[string][]string{"from:spam@evil.test": {"m3"}}}
	e := testEnv(t, box)
	before, err := os.ReadFile(e.paths().ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	e.Stdout, e.Stderr = &out, &errb
	if code := e.runCode([]string{"tidy"}); code != 0 {
		t.Fatalf("tidy %d %s %s", code, out.String(), errb.String())
	}
	if strings.Join(box.trashed, ",") != "m3,m3" {
		t.Fatalf("trashed %v", box.trashed)
	}
	if len(box.ones) != 0 || len(box.reads) != 0 {
		t.Fatalf("ones %v reads %v", box.ones, box.reads)
	}
	text := out.String()
	if !strings.Contains(text, "gmail tidy complete: presets=2 trashed=2") {
		t.Fatalf("summary %s", text)
	}
	if strings.Contains(text, "processed group") || strings.Contains(text, "spam sender groups") || strings.Contains(text, "preset=1 email") {
		t.Fatalf("progress %s", text)
	}
	barLine, _, _ := strings.Cut(text, "\n")
	if barLine != strings.Repeat("█", barCells)+"  2/2" || strings.Contains(barLine, "@") || strings.Contains(text, "\r") {
		t.Fatalf("bar %s", text)
	}
	table := tidyTable([]tidyRow{{"1", "a@example.com", 1}, {"2", "b@example.com", 1}})
	if !strings.Contains(text, table) {
		t.Fatalf("table %s", text)
	}
	if strings.Contains(out.String(), "clickup") || strings.Contains(strings.Join(box.queries, "\n"), "clickup") {
		t.Fatalf("hardcoded query %s queries %v", out.String(), box.queries)
	}
	after, err := os.ReadFile(e.paths().ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("config changed\n%s", after)
	}
}

func TestTidyTableAlignsColumns(t *testing.T) {
	got := tidyTable([]tidyRow{{"1", "a@example.com", 1}, {"2", "b@example.com", 12}})
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != 6 || !strings.HasPrefix(lines[0], "┌") || !strings.HasPrefix(lines[5], "└") {
		t.Fatalf("shape %q", got)
	}
	emailCol := strings.Index(lines[1], "email")
	trashedCol := strings.Index(lines[1], "trashed")
	trashedEnd := trashedCol + len("trashed")
	if emailCol < 0 || trashedCol < 0 {
		t.Fatalf("header %q", lines[1])
	}
	if lines[3][emailCol:emailCol+len("a@example.com")] != "a@example.com" || lines[3][trashedEnd-1] != '1' {
		t.Fatalf("row %q", lines[3])
	}
	if lines[4][emailCol:emailCol+len("b@example.com")] != "b@example.com" || lines[4][trashedEnd-2:trashedEnd] != "12" {
		t.Fatalf("row %q", lines[4])
	}
}

func TestTidyLeavesUnlistedMail(t *testing.T) {
	box := &fakeBox{ids: map[string][]string{"from:other@bulk.test": {"m9"}}}
	out, err, code := run(t, box, []string{"tidy"})
	if code != 0 {
		t.Fatalf("tidy %d %s %s", code, out, err)
	}
	if len(box.trashed) != 0 || strings.Contains(strings.Join(box.queries, "\n"), "clickup") {
		t.Fatalf("trashed %v queries %v", box.trashed, box.queries)
	}
	if !strings.Contains(out, "gmail tidy complete: presets=2 trashed=0") {
		t.Fatalf("summary %s", out)
	}
}

func TestTidyAuditRanksUnreadSenders(t *testing.T) {
	msgs := []api.Message{
		{ID: "a1", Headers: map[string]string{"from": "A <a@bulk.test>"}},
		{ID: "a2", Headers: map[string]string{"from": "A <a@bulk.test>"}},
		{ID: "a3", Headers: map[string]string{"from": "A <a@bulk.test>"}},
		{ID: "b1", Headers: map[string]string{"from": "B <b@bulk.test>"}},
		{ID: "s1", Headers: map[string]string{"from": "Spam <spam@evil.test>"}},
		{ID: "s2", Headers: map[string]string{"from": "Spam <spam@evil.test>"}},
		{ID: "blank", Headers: map[string]string{"from": "nobody"}},
	}
	box := &fakeBox{lists: map[string][]api.Message{unreadQuery: msgs}}
	out, err, code := run(t, box, []string{"tidy", "audit"})
	if code != 0 {
		t.Fatalf("audit %d %s %s", code, out, err)
	}
	if len(box.trashed) != 0 || len(box.ones) != 0 {
		t.Fatalf("audit changed mail trashed %v ones %v", box.trashed, box.ones)
	}
	a := strings.Index(out, "1\ta@bulk.test\tunread=3")
	b := strings.Index(out, "1\tb@bulk.test\tunread=1")
	if a < 0 || b < 0 || a > b {
		t.Fatalf("order %s", out)
	}
	if strings.Contains(out, "spam@evil.test") {
		t.Fatalf("listed sender leaked %s", out)
	}
	if !strings.Contains(out, "gmail tidy audit complete: presets=2 senders=4") {
		t.Fatalf("summary %s", out)
	}
}

func TestTidyAuditRejectsExtraArgs(t *testing.T) {
	_, err, code := run(t, &fakeBox{}, []string{"tidy", "now"})
	if code != 2 || !strings.Contains(err, "c3po gmail tidy") {
		t.Fatalf("usage %d %s", code, err)
	}
}
