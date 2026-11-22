package desk

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/gcal/internal/errs"
	"github.com/ryangerardwilson/c3po/gcal/internal/event"
)

// fakeCalendar is an in-memory primary calendar that records each call.
type fakeCalendar struct {
	upcoming []event.Event
	events   map[string]event.Event
	calls    []string
}

func (f *fakeCalendar) record(format string, args ...any) {
	f.calls = append(f.calls, fmt.Sprintf(format, args...))
}

func (f *fakeCalendar) count(prefix string) int {
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

func (f *fakeCalendar) ListUpcoming(count int, now time.Time) ([]event.Event, error) {
	f.record("list %d %s", count, now.Format(time.RFC3339))
	return f.upcoming, nil
}

func (f *fakeCalendar) Get(id string) (event.Event, error) {
	f.record("get %s", id)
	if ev, ok := f.events[id]; ok {
		return ev, nil
	}
	return event.Event{}, errs.API("Calendar get failed: not found")
}

func (f *fakeCalendar) Delete(id string) error {
	f.record("delete %s", id)
	return nil
}

func (f *fakeCalendar) Create(title string, start, end time.Time, timezone string, invitees []string, requestID string) (event.Event, error) {
	f.record("create %s %s %s %s %s %s", title, start.Format(time.RFC3339), end.Format(time.RFC3339), timezone, strings.Join(invitees, ","), requestID)
	return event.Event{ID: "evt-new", HangoutLink: "https://meet.google.com/abc-defg-hij"}, nil
}

func (f *fakeCalendar) Reschedule(id string, start, end time.Time, timezone string) (event.Event, error) {
	f.record("reschedule %s %s %s %s", id, start.Format(time.RFC3339), end.Format(time.RFC3339), timezone)
	return event.Event{ID: id}, nil
}

// fakeDrive maps file ids to their exported text.
type fakeDrive map[string]string

func (f fakeDrive) ExportText(fileID string) (string, error) {
	text, ok := f[fileID]
	if !ok {
		return "", errs.API("Transcript export failed: not found")
	}
	return text, nil
}

// fakeGoogle is the Connector, Authorizer, and Tokens of the desk tests.
type fakeGoogle struct {
	calendar *fakeCalendar
	drive    fakeDrive
	opened   []string
	email    string
	tokens   map[string]bool
}

func (f *fakeGoogle) Calendar(email, clientSecretFile string) (Calendar, error) {
	f.opened = append(f.opened, "calendar "+email)
	return f.calendar, nil
}

func (f *fakeGoogle) Drive(email, clientSecretFile string) (Drive, error) {
	f.opened = append(f.opened, "drive "+email)
	return f.drive, nil
}

func (f *fakeGoogle) Authorize(clientSecretFile string) (string, error) {
	f.opened = append(f.opened, "authorize "+clientSecretFile)
	return f.email, nil
}

func (f *fakeGoogle) Exists(email string) (bool, error) { return f.tokens[email], nil }

// harness is a temp HOME with XDG dirs and in-memory Google ports.
type harness struct {
	t      *testing.T
	home   string
	vars   map[string]string
	google *fakeGoogle
	stdout strings.Builder
	stderr strings.Builder
	stdin  string
	now    time.Time
	tty    bool
	edited [][]string
	edit   func(argv []string) error
}

func newHarness(t *testing.T) *harness {
	home := t.TempDir()
	h := &harness{t: t, home: home, now: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	h.google = &fakeGoogle{
		calendar: &fakeCalendar{events: map[string]event.Event{}},
		drive:    fakeDrive{}, email: "user@example.com", tokens: map[string]bool{},
	}
	h.vars = map[string]string{
		"HOME":            home,
		"XDG_CONFIG_HOME": filepath.Join(home, "cfg"),
		"XDG_DATA_HOME":   filepath.Join(home, "data"),
		"NO_COLOR":        "1",
	}
	for k, v := range h.vars {
		t.Setenv(k, v)
	}
	h.edit = func([]string) error { return nil }
	return h
}

func (h *harness) configPath() string {
	return filepath.Join(h.home, "cfg", "c3po", "gcal", "config.json")
}

func (h *harness) write(path, body string) {
	h.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		h.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		h.t.Fatal(err)
	}
}

func (h *harness) clientSecret() string {
	p := filepath.Join(h.home, "client_secret.json")
	if _, err := os.Stat(p); err != nil {
		h.write(p, `{"installed":{"client_id":"cid","client_secret":"csecret"}}`)
	}
	return p
}

func (h *harness) writeConfig(timezone string) {
	h.write(h.configPath(), `{"accounts":{"1":{"email":"user@example.com","client_secret_file":"`+h.clientSecret()+`"}},"defaults":{"timezone":"`+timezone+`"}}`)
}

func (h *harness) env() *Env {
	return &Env{
		Lookup: func(k string) (string, bool) {
			v, ok := h.vars[k]
			return v, ok
		},
		Stdin:  bufio.NewReader(strings.NewReader(h.stdin)),
		Stdout: &h.stdout,
		Stderr: &h.stderr,
		Now:    func() time.Time { return h.now },
		NewID:  func() string { return "req-1" },
		TTY:    h.tty,
		Editor: EditorFunc(func(argv []string) error {
			h.edited = append(h.edited, argv)
			return h.edit(argv)
		}),
		Google:     h.google,
		Authorizer: h.google,
		Tokens:     h.google,
	}
}

func (h *harness) run(args ...string) int {
	h.stdout.Reset()
	h.stderr.Reset()
	return Run(args, h.env())
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
