package gmail

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/gmail/internal/config"
)

func TestCleanBarUpdatesInPlace(t *testing.T) {
	var out bytes.Buffer
	bar := cleanBar{out: &out, label: "1  a@example.com", total: 4, live: true, color: true}
	bar.start()
	bar.step(1)
	bar.step(4)
	bar.finish(2)
	got := out.String()
	if strings.Count(got, "\r") != 4 || strings.Count(got, "\n") != 1 {
		t.Fatalf("redraws %q", got)
	}
	if !strings.Contains(got, "█") || !strings.Contains(got, "4/4") || !strings.Contains(got, "trashed 2") {
		t.Fatalf("bar %q", got)
	}
	if !strings.Contains(got, "\x1b[38;5;42m") || !strings.Contains(got, "\x1b[38;5;238m") || !strings.Contains(got, "\x1b[0m") {
		t.Fatalf("color %q", got)
	}
}

func TestCleanBarOnAPipeIsOneLine(t *testing.T) {
	var out bytes.Buffer
	bar := cleanBar{out: &out, label: "1  a@example.com", total: 3, live: false, color: false}
	bar.start()
	bar.step(1)
	bar.step(2)
	bar.finish(0)
	got := out.String()
	if strings.Contains(got, "\r") || strings.Contains(got, "processed group") || strings.Count(got, "\n") != 1 {
		t.Fatalf("pipe %q", got)
	}
	if !strings.Contains(got, "a@example.com") || !strings.Contains(got, "3/3") || !strings.Contains(got, "trashed 0") {
		t.Fatalf("line %q", got)
	}
	if strings.Count(got, "█") != barCells || strings.Contains(got, "░") {
		t.Fatalf("full %q", got)
	}
}

func TestCleanBarShowsOneCellForTheFirstGroup(t *testing.T) {
	var out bytes.Buffer
	bar := cleanBar{out: &out, label: "1  a@example.com", total: 55, live: true, color: false}
	bar.start()
	bar.step(1)
	got := out.String()
	last := got[strings.LastIndex(got, "\r"):]
	if strings.Count(last, "█") != 1 || strings.Count(last, "░") != barCells-1 {
		t.Fatalf("partial %q", got)
	}
	if !strings.Contains(got, "1/55") || strings.Contains(got, "trashed") || strings.Contains(got, "\n") {
		t.Fatalf("counts %q", got)
	}
}

func TestCleanBarBreaksTheLineOnError(t *testing.T) {
	var live, pipe bytes.Buffer
	cleanBar{out: &live, label: "1  a@example.com", total: 2, live: true}.fail()
	cleanBar{out: &pipe, label: "1  a@example.com", total: 2, live: false}.fail()
	if !strings.HasSuffix(live.String(), "\n") || pipe.Len() != 0 {
		t.Fatalf("live %q pipe %q", live.String(), pipe.String())
	}
}

func TestBarColorFollowsTheTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	if !barColor(true) || barColor(false) {
		t.Fatal("plain terminal")
	}
	t.Setenv("NO_COLOR", "1")
	if barColor(true) {
		t.Fatal("NO_COLOR")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if barColor(true) {
		t.Fatal("dumb")
	}
}

func TestNewCleanBarOnABufferIsNotLive(t *testing.T) {
	var out bytes.Buffer
	bar := newCleanBar(&out, "1  a@example.com", 1)
	if bar.live || bar.color || bar.total != 1 {
		t.Fatalf("bar %+v", bar)
	}
}

func TestCleanupEmptySendersIsSilent(t *testing.T) {
	var out bytes.Buffer
	n, err := cleanup(&fakeBox{}, config.Account{Preset: "1", Email: "a@example.com"}, &out)
	if err != nil || n != 0 || out.Len() != 0 {
		t.Fatalf("n %d err %v out %q", n, err, out.String())
	}
}

type failList struct{ *fakeBox }

func (failList) ListIDs(string) ([]string, error) {
	return nil, errors.New("list failed")
}

func TestCleanupReportsAListError(t *testing.T) {
	var out bytes.Buffer
	acct := config.Account{Preset: "1", Email: "a@example.com", SpamSenders: []string{"spam@evil.test"}}
	n, err := cleanup(failList{&fakeBox{}}, acct, &out)
	if err == nil || n != 0 || out.Len() != 0 || strings.Contains(out.String(), "processed group") {
		t.Fatalf("n %d err %v out %q", n, err, out.String())
	}
}
