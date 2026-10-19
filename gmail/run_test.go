package gmail

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/gmail/internal/api"
	"github.com/ryangerardwilson/c3po/gmail/internal/config"
)

func TestHelpVersionAndUsage(t *testing.T) {
	out, err, code := run(t, nil, nil)
	if code != 0 || !strings.Contains(out, "c3po gmail") {
		t.Fatalf("help code=%d out=%q err=%q", code, out, err)
	}
	out, _, code = run(t, nil, []string{"help"})
	if code != 0 || !strings.HasPrefix(out, "c3po gmail") {
		t.Fatalf("help: %d %q", code, out)
	}
	out, _, code = run(t, nil, []string{"version"})
	if code != 0 || !strings.Contains(out, "c3po 0.5.0") {
		t.Fatalf("version: %d %q", code, out)
	}
	_, err, code = run(t, nil, []string{"upgrade"})
	if code != 1 || !strings.Contains(err, "upgrade is not part of c3po gmail. Upgrade c3po instead.") {
		t.Fatalf("upgrade: %d %q", code, err)
	}
	_, err, code = run(t, nil, []string{"1", "ls"})
	if code != 2 || !strings.Contains(err, "error:") {
		t.Fatalf("legacy: %d %q", code, err)
	}
	_, err, code = run(t, nil, []string{"help", "extra"})
	if code != 2 {
		t.Fatalf("help extra: %d %q", code, err)
	}
}

func TestSpamCleanAllPresets(t *testing.T) {
	box := &fakeBox{ids: map[string][]string{
		"from:spam@evil.test": {"m3"},
		"from:clickup.com":    {"m1"},
		"from:mail.clickup.com": {"m1", "m2"},
	}}
	out, err, code := run(t, box, []string{"spam", "clean"})
	if code != 0 {
		t.Fatalf("code %d err %s out %s", code, err, out)
	}
	if !strings.Contains(out, "preset=1 email=a@example.com") || !strings.Contains(out, "preset=2 email=b@example.com") {
		t.Fatalf("banners missing: %s", out)
	}
	if !strings.Contains(out, "spam clean complete: presets=2 trashed_spam=6") {
		t.Fatalf("summary: %s", out)
	}
	if !strings.Contains(out, "1\ttrashed_spam=3") || !strings.Contains(out, "2\ttrashed_spam=3") {
		t.Fatalf("per preset: %s", out)
	}
	if strings.Join(box.trashed, ",") != "m1,m2,m3,m1,m2,m3" {
		t.Fatalf("trashed %v", box.trashed)
	}
}

func TestListQueryAndAccounts(t *testing.T) {
	box := &fakeBox{lists: map[string][]api.Message{
		"is:unread -in:sent from:geeta newer_than:14d": {{
			ID: "abc", ThreadID: "thr", Internal: 10,
			Headers: map[string]string{"from": "Geeta <geeta@example.com>", "subject": "Hello", "date": "Fri, 10 Jan 2025 10:00:00 +0000"},
		}},
	}}
	out, err, code := run(t, box, []string{"1", "list", "unread", "from", "geeta", "since", "2w", "limit", "10"})
	if code != 0 || !strings.Contains(out, "message_id: abc") {
		t.Fatalf("list %d %s %s queries %v", code, out, err, box.queries)
	}
	if box.limits[0] != 10 {
		t.Fatalf("limit %v", box.limits)
	}
	out, _, code = run(t, box, []string{"accounts", "list"})
	if code != 0 || !strings.Contains(out, "token=present") || strings.Contains(out, "super-secret-token") {
		t.Fatalf("accounts: %s", out)
	}
}

func TestSendMarkAndTimer(t *testing.T) {
	box := &fakeBox{sentID: "mid", sentThread: "tid"}
	out, err, code := run(t, box, []string{"1", "send", "to", "boss", "subject", "Notes", "body", "Hello"})
	if code != 0 || !strings.Contains(out, "sent message_id=mid thread_id=tid") {
		t.Fatalf("send %d %s %s", code, out, err)
	}
	if !strings.Contains(string(box.raw), "boss@example.com") || !strings.Contains(string(box.raw), "Notes") {
		t.Fatalf("raw %s", box.raw)
	}
	out, _, code = run(t, box, []string{"1", "preview", "send", "to", "boss", "subject", "Notes", "body", "Hello"})
	if code != 0 || !strings.Contains(out, "side_effects=none") || len(box.raw) == 0 {
		t.Fatalf("preview %d %s", code, out)
	}
	box.modify = "mid thr"
	out, _, code = run(t, box, []string{"1", "mark", "message", "mid", "read"})
	if code != 0 || !strings.Contains(out, "marked_read message_id=mid thread_id=thr") {
		t.Fatalf("mark %s", out)
	}
}

func TestTimerUnitUsesC3PO(t *testing.T) {
	home := t.TempDir()
	var calls []string
	e := testEnv(t, nil)
	e.Home = home
	e.Systemctl = func(args []string) (string, error) {
		calls = append(calls, strings.Join(args, " "))
		return "active\n", nil
	}
	var out, err bytes.Buffer
	e.Stdout, e.Stderr = &out, &err
	if code := e.runCode([]string{"timer", "install"}); code != 0 {
		t.Fatalf("install %d %s", code, err.String())
	}
	body, readErr := os.ReadFile(filepath.Join(home, ".config", "systemd", "user", "gmail.service"))
	if readErr != nil || !strings.Contains(string(body), "gmail spam clean") {
		t.Fatalf("unit %s %v", body, readErr)
	}
	if strings.Join(calls, "|") != "daemon-reload|enable gmail.timer|restart gmail.timer" {
		t.Fatalf("calls %v", calls)
	}
}

type fakeBox struct {
	ids        map[string][]string
	lists      map[string][]api.Message
	queries    []string
	limits     []int
	trashed    []string
	raw        []byte
	sentID     string
	sentThread string
	modify     string
}

func (f *fakeBox) List(query string, limit int) ([]api.Message, error) {
	f.queries = append(f.queries, query)
	f.limits = append(f.limits, limit)
	return f.lists[query], nil
}
func (f *fakeBox) ListIDs(query string) ([]string, error) {
	f.queries = append(f.queries, query)
	return append([]string(nil), f.ids[query]...), nil
}
func (f *fakeBox) ListPage(string, int, string) ([]api.Message, string, error) {
	return nil, "", nil
}
func (f *fakeBox) Get(id string) (api.Message, error) {
	return api.Message{ID: id, Headers: map[string]string{"from": "Ada <ada@example.com>", "subject": "Hi"}}, nil
}
func (f *fakeBox) Meta(id string) (api.Message, error) { return f.Get(id) }
func (f *fakeBox) Thread(id string) ([]api.Message, error) {
	return []api.Message{{ID: id, ThreadID: id}}, nil
}
func (f *fakeBox) Send(raw []byte, _ string) (string, string, error) {
	f.raw = append([]byte(nil), raw...)
	return f.sentID, f.sentThread, nil
}
func (f *fakeBox) TrashIDs(ids []string) (int, error) {
	f.trashed = append(f.trashed, ids...)
	return len(ids), nil
}
func (f *fakeBox) TrashOne(string) error { return nil }
func (f *fakeBox) Modify(id string, _, _ []string) (string, string, error) {
	parts := strings.Split(f.modify, " ")
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}
	return id, "", nil
}
func (f *fakeBox) BatchModify(ids, _, _ []string) (int, error) { return len(ids), nil }
func (f *fakeBox) Attachment(string, string) ([]byte, error)     { return nil, nil }
func (f *fakeBox) UnreadSenders(func(string, int, int)) (map[string]int, error) {
	return map[string]int{}, nil
}

func run(t *testing.T, box *fakeBox, args []string) (string, string, int) {
	t.Helper()
	e := testEnv(t, box)
	var out, err bytes.Buffer
	e.Stdout, e.Stderr = &out, &err
	code := e.runCode(args)
	return out.String(), err.String(), code
}

func testEnv(t *testing.T, box *fakeBox) Env {
	t.Helper()
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret.json")
	sig := filepath.Join(dir, "sig.txt")
	os.WriteFile(secret, []byte(`{"installed":{"client_id":"id"}}`), 0o600)
	os.WriteFile(sig, []byte("Ryan\n"), 0o600)
	cfg := filepath.Join(dir, "config.json")
	body := `{
  "accounts": {
    "1": {"email":"a@example.com","client_secret_file":"` + secret + `","signature_file":"` + sig + `","spam_senders":["spam@evil.test"],"spam_excludes":[],"contacts":{"boss":"boss@example.com"}},
    "2": {"email":"b@example.com","client_secret_file":"` + secret + `","signature_file":"` + sig + `","spam_senders":["spam@evil.test"],"spam_excludes":[],"contacts":{}}
  },
  "defaults": {"list_limit": 10, "timezone_offset": "+05:30"}
}`
	os.WriteFile(cfg, []byte(body), 0o600)
	home := filepath.Join(dir, "home")
	os.MkdirAll(filepath.Join(home, ".local", "share", "c3po", "gmail", "tokens"), 0o700)
	os.WriteFile(filepath.Join(home, ".local", "share", "c3po", "gmail", "tokens", "a@example.com.json"), []byte(`{"token":"super-secret-token"}`), 0o600)
	os.WriteFile(filepath.Join(home, ".local", "share", "c3po", "gmail", "tokens", "b@example.com.json"), []byte(`{"token":"super-secret-token"}`), 0o600)
	env := Env{
		Paths: config.Paths{Getenv: func(k string) string {
			if k == "GMAIL_CLI_CONFIG" {
				return cfg
			}
			return ""
		}, Home: home},
		Home: home,
	}
	if box != nil {
		env.Box = func(config.Account) (Service, error) { return box, nil }
	}
	return env
}
