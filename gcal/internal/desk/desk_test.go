package desk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/gcal/internal/event"
	"github.com/ryangerardwilson/c3po/internal/version"
)

func TestHelpIsGrayOnTerminal(t *testing.T) {
	h := newHarness(t)
	delete(h.vars, "NO_COLOR")
	h.tty = true
	h.run("help")
	if !strings.HasPrefix(h.stdout.String(), ansiGray) || !strings.HasSuffix(h.stdout.String(), ansiReset+"\n") {
		t.Fatalf("not muted: %q", h.stdout.String())
	}
}

func TestVersionAndRemovedCommands(t *testing.T) {
	h := newHarness(t)
	if out := h.mustRun("version"); out != version.String()+"\n" {
		t.Fatalf("version %q", out)
	}
	if msg := h.mustFail(2, "version", "x"); msg != "Use: c3po gcal version\n" {
		t.Fatalf("got %q", msg)
	}
	h.mustFail(2, "help", "x")
	h.mustFail(2, "upgrade")
}

func TestPresetMustBeNumeric(t *testing.T) {
	h := newHarness(t)
	if msg := h.mustFail(2, "abc"); !strings.Contains(msg, "preset must be numeric") {
		t.Fatal(msg)
	}
	if msg := h.mustFail(2, "1"); msg != "Use: c3po gcal <preset> list limit <n>\n" {
		t.Fatal(msg)
	}
}

func TestMissingConfig(t *testing.T) {
	h := newHarness(t)
	msg := h.mustFail(2, "1", "list", "limit", "5")
	if !strings.HasPrefix(msg, "Config not found: "+h.configPath()) {
		t.Fatal(msg)
	}
}

func TestUnknownPreset(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("UTC")
	if msg := h.mustFail(2, "7", "list", "limit", "1"); msg != "Preset `7` not found. Available presets: 1\n" {
		t.Fatal(msg)
	}
}

func TestListKeepsNonRecurringEvents(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("Asia/Kolkata")
	title := "Interview"
	h.google.calendar.upcoming = []event.Event{
		{ID: "evt1", Summary: &title, Start: event.Time{DateTime: "2026-03-10T08:30:00Z"}, End: event.Time{DateTime: "2026-03-10T09:30:00Z"},
			Attendees: []event.Attendee{{Email: "a@example.com"}, {Email: " "}}, HangoutLink: "https://meet.google.com/abc-defg-hij"},
		{ID: "evt2", Start: event.Time{DateTime: "2026-03-11T08:30:00Z"}, End: event.Time{DateTime: "2026-03-11T09:30:00Z"}, RecurringEventID: "s1"},
		{ID: "evt3", Recurrence: []string{"RRULE:FREQ=DAILY"}},
	}
	out := h.mustRun("1", "list", "limit", "5")
	want := "[1]" + strings.Repeat("-", 76) + "\n" +
		"event_id  : evt1\ntitle     : Interview\ntime      : 2026-03-10 14:00:00 -> 2026-03-10 15:00:00\n" +
		"attendees : a@example.com\nhttps://meet.google.com/abc-defg-hij\n"
	if out != want {
		t.Fatalf("got\n%s\nwant\n%s", out, want)
	}
	if got := strings.Join(h.google.calendar.calls, "|"); got != "list 5 2026-03-01T09:00:00Z" {
		t.Fatalf("calls %q", got)
	}
	if got := strings.Join(h.google.opened, "|"); got != "calendar user@example.com" {
		t.Fatalf("opened %q", got)
	}
}

func TestListRequiresCount(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("UTC")
	cases := []struct {
		want string
		args []string
	}{
		{listShape + "\n", []string{"1", "list"}},
		{"limit must be a positive integer\n", []string{"1", "list", "limit", "x"}},
		{"limit must be > 0\n", []string{"1", "list", "limit", "0"}},
		{listShape + "\n", []string{"1", "list", "limit", "5", "output", "text"}},
	}
	for _, tc := range cases {
		if msg := h.mustFail(2, tc.args...); msg != tc.want {
			t.Errorf("%v: %q", tc.args, msg)
		}
	}
	h.mustRun("1", "list", "limit", " +1_0 ")
	if n := h.google.calendar.count("list 10 "); n != 1 {
		t.Fatalf("calls %v", h.google.calendar.calls)
	}
}

func TestCreateValidatesBeforeWriting(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("+05:30")
	cases := map[string][]string{
		"event end must be after event start\n":          {"1", "create", "event", "X", "from", "2026-03-10 14:00:00", "to", "2026-03-10 14:00:00", "invite", "a@x"},
		"time must use YYYY-MM-DD HH:MM:SS\n":            {"1", "create", "event", "X", "from", "2026-02-30 14:00:00", "to", "2026-03-10 14:00:00", "invite", "a@x"},
		"invitees_csv must include at least one email\n": {"1", "create", "event", "X", "from", "2026-03-10 14:00:00", "to", "2026-03-10 15:00:00", "invite", " , "},
		createShape + "\n":                               {"1", "create", "event", "X"},
		actionShape + "\n":                               {"1", "frobnicate"},
	}
	for want, args := range cases {
		if msg := h.mustFail(2, args...); msg != want {
			t.Errorf("%v: %q", args, msg)
		}
	}
	if n := h.google.calendar.count("create "); n != 0 {
		t.Fatalf("%d creates", n)
	}
}

func TestCreateSendsParsedWindow(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("Asia/Kolkata")
	out := h.mustRun("1", "create", "event", "Interview", "from", "2026-03-10 14:00:00", "to", "2026-03-10 15:00:00", "invite", "a@example.com, b@example.com,")
	if out != "created\tevent_id=evt-new\nhttps://meet.google.com/abc-defg-hij\n" {
		t.Fatalf("out %q", out)
	}
	want := "create Interview 2026-03-10T14:00:00+05:30 2026-03-10T15:00:00+05:30 Asia/Kolkata a@example.com,b@example.com req-1"
	if got := h.google.calendar.calls[0]; got != want {
		t.Fatalf("call %q", got)
	}
}

func TestPreviewOpensNoGoogleService(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("Asia/Kolkata")
	h.mustRun("1", "preview", "create", "event", "X", "from", "2026-03-10 14:00:00", "to", "2026-03-10 15:00:00", "invite", "a@x")
	h.mustRun("1", "preview", "reschedule", "event", "abc", "from", "2026-03-11 10:00:00", "to", "2026-03-11 11:00:00")
	if len(h.google.opened) != 0 || len(h.google.calendar.calls) != 0 {
		t.Fatalf("opened %v calls %v", h.google.opened, h.google.calendar.calls)
	}
}

func TestExportOpensDriveBeforeReadingEvent(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("UTC")
	title := "Weekly Sync!"
	h.google.calendar.events["evt1"] = event.Event{ID: "evt1", Summary: &title, Start: event.Time{DateTime: "2026-03-10T14:00:00+05:30"},
		Attachments: []event.Attachment{{FileID: "file1", Title: "Meeting Transcript", MimeType: "application/vnd.google-apps.document"}}}
	h.google.drive["file1"] = "hello"
	out := h.mustRun("1", "export", "event", "evt1")
	path := filepath.Join(h.home, "data", "c3po", "gcal", "transcripts", "2026-03-10_weekly-sync_evt1.txt")
	if out != "saved transcript\tevent_id=evt1\tfile_id=file1\n"+path+"\n" {
		t.Fatalf("out %q", out)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "hello" {
		t.Fatalf("file %q %v", got, err)
	}
	if got := strings.Join(h.google.opened, "|"); got != "calendar user@example.com|drive user@example.com" {
		t.Fatalf("opened %q", got)
	}
}

func TestAccountsAskTheTokenPort(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("UTC")
	if out := h.mustRun("accounts", "list"); out != "account preset=1 email=user@example.com token=missing\n" {
		t.Fatalf("out %q", out)
	}
	h.google.tokens["user@example.com"] = true
	if out := h.mustRun("accounts", "list"); out != "account preset=1 email=user@example.com token=present\n" {
		t.Fatalf("out %q", out)
	}
	if len(h.google.opened) != 0 {
		t.Fatalf("opened %v", h.google.opened)
	}
}

func TestAuthSavesPreset(t *testing.T) {
	h := newHarness(t)
	h.stdin = "+0530\n"
	secret := h.clientSecret()
	out := h.mustRun("auth", secret)
	if out != "Timezone [UTC, +0530, -0430]: authorized\t1\tuser@example.com\t+05:30\n" {
		t.Fatalf("out %q", out)
	}
	if got := strings.Join(h.google.opened, "|"); got != "authorize "+secret {
		t.Fatalf("opened %q", got)
	}
	if out := h.mustRun("setup", "check"); out != "config="+h.configPath()+"\ntimezone=+05:30\naccount preset=1 email=user@example.com token=missing\n" {
		t.Fatalf("setup %q", out)
	}
	h.stdin = ""
	if msg := h.mustFail(1, "auth", secret); msg != "gcal: EOF when reading a line\n" {
		t.Fatalf("msg %q", msg)
	}
	if msg := h.mustFail(2, "auth", filepath.Join(h.home, "nope.json")); !strings.HasPrefix(msg, "missing client secret file: ") {
		t.Fatalf("msg %q", msg)
	}
}

func TestConfigOpensEditor(t *testing.T) {
	h := newHarness(t)
	h.vars["EDITOR"] = "nvim -u 'my init.vim'"
	h.mustRun("config")
	got, _ := os.ReadFile(h.configPath())
	if string(got) != emptyConfig {
		t.Fatalf("config %q", got)
	}
	want := []string{"nvim", "-u", "my init.vim", h.configPath()}
	if strings.Join(h.edited[0], "|") != strings.Join(want, "|") {
		t.Fatalf("argv %q", h.edited[0])
	}
	h.vars["VISUAL"] = "code -w"
	h.mustRun("config")
	if h.edited[1][0] != "code" {
		t.Fatalf("argv %q", h.edited[1])
	}
	h.vars["VISUAL"] = "   "
	if msg := h.mustFail(2, "config"); msg != "Editor command is empty. Set VISUAL or EDITOR.\n" {
		t.Fatal(msg)
	}
	h.mustFail(2, "config", "extra")
}

func TestEditorErrors(t *testing.T) {
	if err := editorError(nil, "vim"); err != nil {
		t.Fatal(err)
	}
	err := editorError(fmt.Errorf("start: %w", ErrEditorNotFound), "/nonexistent/editor-xyz")
	if err == nil || err.Error() != "Editor not found: /nonexistent/editor-xyz. Set VISUAL or EDITOR." {
		t.Fatalf("got %v", err)
	}
	err = editorError(&EditorExitError{Code: 1}, "false")
	if err == nil || err.Error() != "Editor exited with code 1" {
		t.Fatalf("got %v", err)
	}
	other := errors.New("boom")
	if err := editorError(other, "vim"); err != other {
		t.Fatalf("got %v", err)
	}
	h := newHarness(t)
	h.vars["VISUAL"] = "false"
	h.edit = func([]string) error { return &EditorExitError{Code: 3} }
	if msg := h.mustFail(2, "config"); msg != "Editor exited with code 3\n" {
		t.Fatal(msg)
	}
	h.edit = func([]string) error { return other }
	if msg := h.mustFail(1, "config"); msg != "gcal: boom\n" {
		t.Fatal(msg)
	}
}

func TestFormatEventTime(t *testing.T) {
	if got := formatEventTime("2026-03-10", "2026-03-11", "UTC"); got != "2026-03-10 -> 2026-03-11" {
		t.Fatal(got)
	}
	if got := formatEventTime("2026-03-10T14:00:00+05:30", "2026-03-10T15:00:00Z", "UTC"); got != "2026-03-10 08:30:00 -> 2026-03-10 15:00:00" {
		t.Fatal(got)
	}
}
