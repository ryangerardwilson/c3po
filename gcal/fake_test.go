package gcal

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeGoogle serves the token, userinfo, Calendar, and Drive endpoints.
type fakeGoogle struct {
	t        *testing.T
	srv      *httptest.Server
	mu       sync.Mutex
	requests []recorded
	events   map[string]map[string]any
	list     []map[string]any
	email    string
	transcript string
}

type recorded struct {
	Method string
	Path   string
	Query  url.Values
	Form   url.Values
	Body   map[string]any
	Auth   string
}

func newFakeGoogle(t *testing.T) *fakeGoogle {
	f := &fakeGoogle{t: t, events: map[string]map[string]any{}, email: "User@Example.com"}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeGoogle) record(r *http.Request) recorded {
	rec := recorded{Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), Auth: r.Header.Get("Authorization")}
	if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		_ = r.ParseForm()
		rec.Form = r.PostForm
	} else if r.Body != nil {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &rec.Body)
	}
	f.mu.Lock()
	f.requests = append(f.requests, rec)
	f.mu.Unlock()
	return rec
}

func (f *fakeGoogle) find(method, path string) []recorded {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []recorded
	for _, r := range f.requests {
		if r.Method == method && r.Path == path {
			out = append(out, r)
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (f *fakeGoogle) serve(w http.ResponseWriter, r *http.Request) {
	rec := f.record(r)
	switch {
	case rec.Path == "/token":
		f.token(w, rec)
	case rec.Path == "/userinfo":
		writeJSON(w, 200, map[string]any{"email": f.email})
	case rec.Path == "/calendar/calendars/primary/events" && rec.Method == http.MethodGet:
		writeJSON(w, 200, map[string]any{"items": f.list})
	case rec.Path == "/calendar/calendars/primary/events" && rec.Method == http.MethodPost:
		writeJSON(w, 200, map[string]any{"id": "evt-new", "summary": rec.Body["summary"], "hangoutLink": "https://meet.google.com/abc-defg-hij"})
	case strings.HasPrefix(rec.Path, "/calendar/calendars/primary/events/"):
		f.event(w, rec)
	case strings.HasPrefix(rec.Path, "/drive/files/"):
		_, _ = io.WriteString(w, f.transcript)
	default:
		writeJSON(w, 404, map[string]any{"error": map[string]any{"message": "Not Found"}})
	}
}

func (f *fakeGoogle) token(w http.ResponseWriter, rec recorded) {
	switch {
	case rec.Form.Get("grant_type") == "authorization_code" && rec.Form.Get("code") == "good-code":
		writeJSON(w, 200, map[string]any{"access_token": "flow-access", "refresh_token": "flow-refresh", "expires_in": 3599, "token_type": "Bearer"})
	case rec.Form.Get("grant_type") == "refresh_token" && rec.Form.Get("refresh_token") == "good-refresh":
		writeJSON(w, 200, map[string]any{"access_token": "refreshed-access", "expires_in": 3599, "token_type": "Bearer"})
	default:
		writeJSON(w, 400, map[string]any{"error": "invalid_grant"})
	}
}

func (f *fakeGoogle) event(w http.ResponseWriter, rec recorded) {
	id := strings.TrimPrefix(rec.Path, "/calendar/calendars/primary/events/")
	ev, ok := f.events[id]
	switch {
	case !ok:
		writeJSON(w, 404, map[string]any{"error": map[string]any{"message": "Not Found"}})
	case rec.Method == http.MethodDelete:
		w.WriteHeader(204)
	default:
		writeJSON(w, 200, ev)
	}
}

// harness is a temp HOME with XDG dirs, a config, and a fake Google.
type harness struct {
	t      *testing.T
	home   string
	vars   map[string]string
	google *fakeGoogle
	stdout strings.Builder
	stderr strings.Builder
	stdin  string
	now    time.Time
	opened []string
	edited [][]string
	browse func(string) error
}

func newHarness(t *testing.T) *harness {
	home := t.TempDir()
	h := &harness{t: t, home: home, google: newFakeGoogle(t), now: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	h.vars = map[string]string{
		"HOME":            home,
		"XDG_CONFIG_HOME": filepath.Join(home, "cfg"),
		"XDG_DATA_HOME":   filepath.Join(home, "data"),
		"NO_COLOR":        "1",
	}
	h.browse = h.approve("good-code")
	return h
}

func (h *harness) path(rel string) string { return filepath.Join(h.home, rel) }

func (h *harness) configPath() string { return filepath.Join(h.home, "cfg", "c3po", "gcal", "config.json") }

func (h *harness) tokenPath(email string) string {
	return filepath.Join(h.home, "data", "c3po", "gcal", "tokens", email+".json")
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
	p := h.path("client_secret.json")
	if _, err := os.Stat(p); err != nil {
		h.write(p, `{"installed":{"client_id":"cid","client_secret":"csecret","auth_uri":"`+h.google.srv.URL+`/auth","token_uri":"`+h.google.srv.URL+`/token"}}`)
	}
	return p
}

func (h *harness) writeConfig(timezone string) {
	h.write(h.configPath(), `{"accounts":{"1":{"email":"user@example.com","client_secret_file":"`+h.clientSecret()+`"}},"defaults":{"timezone":"`+timezone+`"}}`)
}

// writeToken saves a google-auth style token for user@example.com.
func (h *harness) writeToken(access, refresh string, expiry time.Time, scopes []string) {
	tok := map[string]any{
		"token": access, "refresh_token": refresh, "token_uri": h.google.srv.URL + "/token",
		"client_id": "cid", "client_secret": "csecret", "scopes": scopes,
		"universe_domain": "googleapis.com", "account": "",
		"expiry": expiry.UTC().Format("2006-01-02T15:04:05.000000") + "Z",
	}
	raw, _ := json.Marshal(tok)
	h.write(h.tokenPath("user@example.com"), string(raw))
}

// approve is a browser that grants code at the redirect URI.
func (h *harness) approve(code string) func(string) error {
	return func(authURL string) error {
		h.opened = append(h.opened, authURL)
		u, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		q := u.Query()
		cb := q.Get("redirect_uri") + "?" + url.Values{"code": {code}, "state": {q.Get("state")}}.Encode()
		resp, err := http.Get(cb)
		if err != nil {
			return err
		}
		return resp.Body.Close()
	}
}

func (h *harness) env() *env {
	return &env{
		lookup: func(k string) (string, bool) {
			v, ok := h.vars[k]
			return v, ok
		},
		stdin:        bufio.NewReader(strings.NewReader(h.stdin)),
		stdout:       &h.stdout,
		stderr:       &h.stderr,
		http:         h.google.srv.Client(),
		calendarBase: h.google.srv.URL + "/calendar",
		driveBase:    h.google.srv.URL + "/drive",
		userinfoURL:  h.google.srv.URL + "/userinfo",
		now:          func() time.Time { return h.now },
		openBrowser:  func(u string) error { return h.browse(u) },
		editor: func(argv []string) error {
			h.edited = append(h.edited, argv)
			return nil
		},
		newID: func() string { return "req-1" },
	}
}

func (h *harness) run(args ...string) int {
	h.stdout.Reset()
	h.stderr.Reset()
	return run(args, h.env())
}
