package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScreenPresetsOmitTheToken(t *testing.T) {
	cfg := Config{"accounts": map[string]any{
		"2": map[string]any{"name": "ade", "token": map[string]any{"user": "xoxp-ade-secret"}},
		"1": map[string]any{"name": "work", "token": map[string]any{"user": "xoxp-work-secret"}},
	}}
	presets := ScreenPresets(cfg)
	if len(presets) != 2 || presets[0].ID != "1" || presets[0].Name != "work" || presets[1].Name != "ade" {
		t.Fatalf("%#v", presets)
	}
	for _, preset := range presets {
		if strings.Contains(preset.ID+preset.Name, "xoxp-") {
			t.Fatalf("token in %#v", preset)
		}
	}
}

func TestScreenHistoryIsOldestFirst(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/users.info":
			_, _ = w.Write([]byte(`{"ok":true,"user":{"id":"U2","real_name":"Ada"}}`))
		case "/api/conversations.history":
			_, _ = w.Write([]byte(`{"ok":true,"messages":[{"ts":"200.2","user":"U2","text":"later","thread_ts":"100.1"},{"ts":"100.1","user":"U2","text":"earlier"}]}`))
		default:
			t.Errorf("path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	old := slackAPIBase
	slackAPIBase = server.URL + "/api/"
	defer func() { slackAPIBase = old }()
	client := SlackClient{Token: "xoxp-user", HTTPClient: server.Client()}
	rows, err := ScreenHistory(client, "D1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Text != "earlier" || rows[1].Text != "later" || rows[1].ThreadTS != "100.1" || rows[0].User != "Ada" {
		t.Fatalf("%#v", rows)
	}
}

func TestScreenConversationsSortUnreadFirst(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		types := r.URL.Query().Get("types")
		switch {
		case strings.Contains(types, "im"):
			_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"D1","is_im":true,"user":"U9"}]}`))
		case strings.Contains(types, "public_channel"):
			_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"C1","name":"general","is_channel":true,"unread_count_display":4}]}`))
		case r.URL.Path == "/api/users.info":
			_, _ = w.Write([]byte(`{"ok":true,"user":{"id":"U9","profile":{"display_name":"Ada"}}}`))
		default:
			t.Errorf("path %s types %s", r.URL.Path, types)
		}
	}))
	defer server.Close()
	old := slackAPIBase
	slackAPIBase = server.URL + "/api/"
	defer func() { slackAPIBase = old }()
	client := SlackClient{Token: "xoxp-user", HTTPClient: server.Client()}
	rows, err := ScreenConversations(client)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Name != "general" || rows[0].Unread != 4 || rows[1].Name != "Ada" || rows[1].Surface != "dm" {
		t.Fatalf("%#v", rows)
	}
}
