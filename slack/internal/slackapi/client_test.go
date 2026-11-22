package slackapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRateLimitRetriesWithVerboseProgress(t *testing.T) {
	var sleeps []time.Duration
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/conversations.history" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"error":"ratelimited"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"messages":[{"ts":"100.1","user":"U2","text":"hi"}]}`))
	}))
	defer server.Close()

	var stderr bytes.Buffer
	client := Client{
		Token:      "xoxp-user",
		HTTPClient: server.Client(),
		BaseURL:    server.URL + "/api/",
		Verbose:    &stderr,
		Sleep: func(d time.Duration) {
			sleeps = append(sleeps, d)
		},
	}
	data, err := client.Request("conversations.history", map[string]string{"channel": "C1", "limit": "1"}, false, http.MethodGet)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if data["ok"] != true {
		t.Fatalf("expected success after retry: %#v", data)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if len(sleeps) != 1 || sleeps[0] != time.Second {
		t.Fatalf("unexpected sleeps: %#v", sleeps)
	}
	log := stderr.String()
	if !strings.Contains(log, "rate_limited") || !strings.Contains(log, "retry_after=1s") {
		t.Fatalf("expected rate limit progress on stderr, got: %s", log)
	}
}

func TestListAPIPaginationVerbose(t *testing.T) {
	page := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		if page == 1 {
			_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"C1","name":"one"}],"response_metadata":{"next_cursor":"page2cursor"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"C2","name":"two"}],"response_metadata":{"next_cursor":""}}`))
	}))
	defer server.Close()
	var stderr bytes.Buffer
	client := Client{Token: "xoxp-user", HTTPClient: server.Client(), BaseURL: server.URL + "/api/", Verbose: &stderr}
	rows, err := listAPI(client, "users.conversations", map[string]string{"types": "public_channel", "limit": "1"}, "channels")
	if err != nil {
		t.Fatalf("listAPI: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %#v", rows)
	}
	log := stderr.String()
	if !strings.Contains(log, "paginate method=users.conversations page=1") || !strings.Contains(log, "page=2") {
		t.Fatalf("expected pagination progress, got: %s", log)
	}
}
