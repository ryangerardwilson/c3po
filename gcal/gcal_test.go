package gcal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/gcal/internal/oauth"
	"github.com/ryangerardwilson/c3po/internal/version"
)

func (h *harness) validToken() {
	h.writeToken("valid-access", "good-refresh", h.now.Add(time.Hour), oauth.CalendarScopes)
}

func (h *harness) mustRun(args ...string) string {
	h.t.Helper()
	if code := h.run(args...); code != 0 {
		h.t.Fatalf("%v: exit %d, stderr %q", args, code, h.stderr.String())
	}
	return h.stdout.String()
}

func (h *harness) mustFail(code int, args ...string) string {
	h.t.Helper()
	if got := h.run(args...); got != code {
		h.t.Fatalf("%v: exit %d, want %d, stderr %q", args, got, code, h.stderr.String())
	}
	return h.stderr.String()
}

func TestHelpIsHumanFriendly(t *testing.T) {
	var out, errb strings.Builder
	for _, args := range [][]string{{"help"}, {}, {"-h"}, {"--help"}} {
		out.Reset()
		if code := Run(args, &out, &errb); code != 0 {
			t.Fatalf("%v exit %d", args, code)
		}
		if out.String() != Help {
			t.Fatalf("%v printed %q", args, out.String())
		}
	}
	for _, want := range []string{
		"features:", "authorize a Google account and save or refresh its preset",
		"create a calendar event with invitees and a Google Meet link",
		"# auth <client_secret_path>", "# <preset> list limit <n>",
		"c3po gcal accounts list", "c3po gcal 1 preview create event", "output json",
		"c3po gcal 1 list limit 5",
		`c3po gcal 1 reschedule event abc123 from "2026-03-11 10:00:00" to "2026-03-11 11:00:00"`,
	} {
		if !strings.Contains(Help, want) {
			t.Errorf("help lacks %q", want)
		}
	}
	for _, bad := range []string{"commands:", "usage:", "\n  gcal "} {
		if strings.Contains(Help, bad) {
			t.Errorf("help has %q", bad)
		}
	}
}

func TestHelpIsGrayOnTerminal(t *testing.T) {
	h := newHarness(t)
	delete(h.vars, "NO_COLOR")
	e := h.env()
	e.tty = true
	run([]string{"help"}, e)
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

func TestListCommand(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("Asia/Kolkata")
	h.validToken()
	h.google.list = []map[string]any{
		{"id": "evt1", "summary": "Interview", "start": map[string]any{"dateTime": "2026-03-10T08:30:00Z"},
			"end": map[string]any{"dateTime": "2026-03-10T09:30:00Z"}, "attendees": []any{map[string]any{"email": "a@example.com"}, map[string]any{"email": "b@example.com"}},
			"hangoutLink": "https://meet.google.com/abc-defg-hij"},
		{"id": "evt2", "summary": "Series", "start": map[string]any{"dateTime": "2026-03-11T08:30:00Z"},
			"end": map[string]any{"dateTime": "2026-03-11T09:30:00Z"}, "recurringEventId": "s1"},
		{"id": "evt3", "start": map[string]any{"date": "2026-03-12"}, "end": map[string]any{"date": "2026-03-13"}},
	}
	out := h.mustRun("1", "list", "limit", "5")
	want := "[1]" + strings.Repeat("-", 76) + "\n" +
		"event_id  : evt1\ntitle     : Interview\ntime      : 2026-03-10 14:00:00 -> 2026-03-10 15:00:00\n" +
		"attendees : a@example.com,b@example.com\nhttps://meet.google.com/abc-defg-hij\n" +
		"[2]" + strings.Repeat("-", 76) + "\n" +
		"event_id  : evt3\ntitle     : (untitled)\ntime      : 2026-03-12 -> 2026-03-13\nattendees : -\n-\n"
	if out != want {
		t.Fatalf("got\n%s\nwant\n%s", out, want)
	}
	reqs := h.google.find("GET", "/calendar/calendars/primary/events")
	q := reqs[0].Query
	if q.Get("maxResults") != "5" || q.Get("singleEvents") != "true" || q.Get("orderBy") != "startTime" ||
		q.Get("timeMin") != "2026-03-01T09:00:00Z" || q.Has("conferenceDataVersion") {
		t.Fatalf("query %v", q)
	}
	if reqs[0].Auth != "Bearer valid-access" {
		t.Fatalf("auth %q", reqs[0].Auth)
	}
}

func TestListOutputJSON(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("UTC")
	h.validToken()
	h.google.list = []map[string]any{{"id": "evt1", "summary": "Café ☕", "start": map[string]any{"dateTime": "2026-03-10T14:00:00+05:30"},
		"end": map[string]any{"dateTime": "2026-03-10T15:00:00+05:30"}, "attendees": []any{map[string]any{"email": "a@example.com"}}}}
	out := h.mustRun("1", "list", "limit", "5", "output", "json")
	want := `{
  "events": [
    {
      "attendees": [
        "a@example.com"
      ],
      "event_id": "evt1",
      "meeting_url": "-",
      "time": "2026-03-10 08:30:00 -> 2026-03-10 09:30:00",
      "title": "Caf\u00e9 \u2615"
    }
  ]
}
`
	if out != want {
		t.Fatalf("got\n%s", out)
	}
	h.google.list = nil
	if out := h.mustRun("1", "list", "limit", "2", "output", "json"); out != "{\n  \"events\": []\n}\n" {
		t.Fatalf("empty %q", out)
	}
	if out := h.mustRun("1", "list", "limit", "2"); out != "no events\n" {
		t.Fatalf("empty %q", out)
	}
}

func TestListRequiresCount(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("UTC")
	h.validToken()
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
}

func TestCreateEventCommand(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("Asia/Kolkata")
	h.validToken()
	out := h.mustRun("1", "create", "event", "Interview", "from", "2026-03-10 14:00:00", "to", "2026-03-10 15:00:00", "invite", "a@example.com, b@example.com,")
	if out != "created\tevent_id=evt-new\nhttps://meet.google.com/abc-defg-hij\n" {
		t.Fatalf("out %q", out)
	}
	req := h.google.find("POST", "/calendar/calendars/primary/events")[0]
	if req.Query.Get("conferenceDataVersion") != "1" || req.Query.Get("sendUpdates") != "all" {
		t.Fatalf("query %v", req.Query)
	}
	raw, _ := json.Marshal(req.Body)
	want := `{"attendees":[{"email":"a@example.com"},{"email":"b@example.com"}],"conferenceData":{"createRequest":{"conferenceSolutionKey":{"type":"hangoutsMeet"},"requestId":"req-1"}},"end":{"dateTime":"2026-03-10T15:00:00+05:30","timeZone":"Asia/Kolkata"},"start":{"dateTime":"2026-03-10T14:00:00+05:30","timeZone":"Asia/Kolkata"},"summary":"Interview"}`
	if string(raw) != want {
		t.Fatalf("body %s", raw)
	}
}

func TestCreateValidatesBeforeWriting(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("+05:30")
	h.validToken()
	cases := map[string][]string{
		"event end must be after event start\n":         {"1", "create", "event", "X", "from", "2026-03-10 14:00:00", "to", "2026-03-10 14:00:00", "invite", "a@x"},
		"time must use YYYY-MM-DD HH:MM:SS\n":           {"1", "create", "event", "X", "from", "2026-02-30 14:00:00", "to", "2026-03-10 14:00:00", "invite", "a@x"},
		"invitees_csv must include at least one email\n": {"1", "create", "event", "X", "from", "2026-03-10 14:00:00", "to", "2026-03-10 15:00:00", "invite", " , "},
		createShape + "\n":                               {"1", "create", "event", "X"},
		actionShape + "\n":                               {"1", "frobnicate"},
	}
	for want, args := range cases {
		if msg := h.mustFail(2, args...); msg != want {
			t.Errorf("%v: %q", args, msg)
		}
	}
	if n := len(h.google.find("POST", "/calendar/calendars/primary/events")); n != 0 {
		t.Fatalf("%d creates", n)
	}
}

func TestOffsetTimezoneOmitsTimeZoneField(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("-04:30")
	h.validToken()
	h.mustRun("1", "create", "event", "X", "from", "2026-3-10 9:05:00", "to", "2026-03-10 10:00:00", "invite", "a@x")
	body := h.google.find("POST", "/calendar/calendars/primary/events")[0].Body
	start := body["start"].(map[string]any)
	if start["dateTime"] != "2026-03-10T09:05:00-04:30" || start["timeZone"] != nil {
		t.Fatalf("start %v", start)
	}
}

func TestDeleteCommand(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("UTC")
	h.validToken()
	h.google.events["evt1"] = map[string]any{"id": "evt1"}
	if out := h.mustRun("1", "delete", "event", "evt1"); out != "deleted\tevent_id=evt1\n" {
		t.Fatalf("out %q", out)
	}
	req := h.google.find("DELETE", "/calendar/calendars/primary/events/evt1")[0]
	if req.Query.Get("sendUpdates") != "all" {
		t.Fatalf("query %v", req.Query)
	}
	msg := h.mustFail(2, "1", "delete", "event", "nope")
	if !strings.HasPrefix(msg, "Calendar delete failed: <HttpError 404 when requesting ") || !strings.Contains(msg, `returned "Not Found"`) {
		t.Fatalf("msg %q", msg)
	}
}

func TestRescheduleCommand(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("Asia/Kolkata")
	h.validToken()
	h.google.events["evt1"] = map[string]any{"id": "evt1", "conferenceData": map[string]any{"entryPoints": []any{
		map[string]any{"entryPointType": "phone", "uri": "tel:1"}, map[string]any{"entryPointType": "video", "uri": " https://meet.google.com/x "}}}}
	out := h.mustRun("1", "reschedule", "event", "evt1", "from", "2026-03-11 10:00:00", "to", "2026-03-11 11:00:00")
	if out != "rescheduled\tevent_id=evt1\nhttps://meet.google.com/x\n" {
		t.Fatalf("out %q", out)
	}
	req := h.google.find("PATCH", "/calendar/calendars/primary/events/evt1")[0]
	if req.Query.Get("sendUpdates") != "all" || req.Query.Get("conferenceDataVersion") != "1" {
		t.Fatalf("query %v", req.Query)
	}
	if req.Body["start"].(map[string]any)["dateTime"] != "2026-03-11T10:00:00+05:30" || req.Body["summary"] != nil {
		t.Fatalf("body %v", req.Body)
	}
	h.mustFail(2, "1", "reschedule", "event", "evt1", "from", "x")
}

func TestPreviewDoesNotTouchCredentialsOrNetwork(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("Asia/Kolkata")
	out := h.mustRun("1", "preview", "create", "event", "Ryan's sync", "from", "2026-03-10 14:00:00", "to", "2026-03-10 15:00:00", "invite", "a@example.com,b@example.com")
	want := `preview_create_event title="Ryan's sync" from=2026-03-10T14:00:00+05:30 to=2026-03-10T15:00:00+05:30 timezone=Asia/Kolkata invite=a@example.com,b@example.com meet=yes side_effects=none` + "\n"
	if out != want {
		t.Fatalf("got %q", out)
	}
	out = h.mustRun("1", "preview", "reschedule", "event", "abc123", "from", "2026-03-11 10:00:00", "to", "2026-03-11 11:00:00")
	want = "preview_reschedule_event event_id=abc123 from=2026-03-11T10:00:00+05:30 to=2026-03-11T11:00:00+05:30 timezone=Asia/Kolkata side_effects=none\n"
	if out != want {
		t.Fatalf("got %q", out)
	}
	h.mustFail(2, "1", "preview")
	h.mustFail(2, "1", "preview", "delete")
	if len(h.google.requests) != 0 || len(h.opened) != 0 {
		t.Fatalf("network used: %v", h.google.requests)
	}
}

func TestExportTranscriptCommand(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("UTC")
	h.writeToken("valid-access", "good-refresh", h.now.Add(time.Hour), oauth.TranscriptScopes)
	h.google.transcript = "hello\xff"
	h.google.events["evt1"] = map[string]any{"id": "evt1", "summary": "Weekly Sync!", "start": map[string]any{"dateTime": "2026-03-10T14:00:00+05:30"},
		"attachments": []any{
			map[string]any{"fileId": "notes", "title": "Notes", "mimeType": "application/vnd.google-apps.document"},
			map[string]any{"fileId": "file1", "title": "Meeting Transcript", "fileUrl": "https://docs.google.com/document/d/file1"},
		}}
	out := h.mustRun("1", "export", "event", "evt1")
	path := filepath.Join(h.home, "data", "c3po", "gcal", "transcripts", "2026-03-10_weekly-sync_evt1.txt")
	if out != "saved transcript\tevent_id=evt1\tfile_id=file1\n"+path+"\n" {
		t.Fatalf("out %q", out)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "hello�" {
		t.Fatalf("file %q %v", got, err)
	}
	if q := h.google.find("GET", "/drive/files/file1/export")[0].Query; q.Get("mimeType") != "text/plain" {
		t.Fatalf("query %v", q)
	}
	h.google.events["evt2"] = map[string]any{"id": "evt2"}
	if msg := h.mustFail(2, "1", "export", "event", "evt2"); msg != "No transcript attachment found for this event.\n" {
		t.Fatal(msg)
	}
	h.mustFail(2, "1", "export", "evt1")
}

func TestExportAsksForDriveScopeThroughRefresh(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("UTC")
	h.validToken()
	h.google.transcript = "hi"
	h.google.events["evt1"] = map[string]any{"id": "evt1", "attachments": []any{
		map[string]any{"fileId": "f", "title": "Transcript", "mimeType": "application/vnd.google-apps.document"}}}
	h.mustRun("1", "export", "event", "evt1")
	refresh := h.google.find("POST", "/token")
	if len(refresh) != 1 || refresh[0].Form.Get("scope") != strings.Join(oauth.TranscriptScopes, " ") {
		t.Fatalf("refresh %v", refresh)
	}
	if auth := h.google.find("GET", "/drive/files/f/export")[0].Auth; auth != "Bearer refreshed-access" {
		t.Fatalf("drive auth %q", auth)
	}
	if len(h.opened) != 0 {
		t.Fatal("browser opened")
	}
}

func TestAccountsListAndSetupCheck(t *testing.T) {
	h := newHarness(t)
	h.writeConfig("Asia/Kolkata")
	if out := h.mustRun("accounts", "list"); out != "account preset=1 email=user@example.com token=missing\n" {
		t.Fatalf("out %q", out)
	}
	h.validToken()
	out := h.mustRun("accounts", "list", "output", "json")
	if out != "{\n  \"accounts\": [\n    {\n      \"email\": \"user@example.com\",\n      \"preset\": \"1\",\n      \"token\": \"present\"\n    }\n  ]\n}\n" {
		t.Fatalf("json %q", out)
	}
	out = h.mustRun("setup", "check")
	if out != "config="+h.configPath()+"\ntimezone=Asia/Kolkata\naccount preset=1 email=user@example.com token=present\n" {
		t.Fatalf("setup %q", out)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(h.mustRun("setup", "check", "output", "json")), &payload); err != nil || payload["config"] != h.configPath() {
		t.Fatalf("setup json %v %v", payload, err)
	}
	h.mustFail(2, "accounts")
	h.mustFail(2, "accounts", "list", "json")
	h.mustFail(2, "setup", "now")
	if len(h.google.requests) != 0 {
		t.Fatal("network used")
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
	err := editorError(runEditor([]string{"/nonexistent/editor-xyz"}, os.Stdout, os.Stderr), "/nonexistent/editor-xyz")
	if err == nil || err.Error() != "Editor not found: /nonexistent/editor-xyz. Set VISUAL or EDITOR." {
		t.Fatalf("got %v", err)
	}
	err = editorError(runEditor([]string{"false"}, os.Stdout, os.Stderr), "false")
	if err == nil || err.Error() != "Editor exited with code 1" {
		t.Fatalf("got %v", err)
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
