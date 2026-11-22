package app

import (
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
	slack := &fakeSlack{
		t:     t,
		users: map[string]map[string]any{"U2": {"id": "U2", "real_name": "Ada"}},
		history: map[string][]map[string]any{"D1": {
			{"ts": "200.2", "user": "U2", "text": "later", "thread_ts": "100.1"},
			{"ts": "100.1", "user": "U2", "text": "earlier"},
		}},
	}
	client, err := ScreenClient(slack, Account{"token": map[string]any{"user": "xoxp-user"}})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ScreenHistory(client, "D1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Text != "earlier" || rows[1].Text != "later" || rows[1].ThreadTS != "100.1" || rows[0].User != "Ada" {
		t.Fatalf("%#v", rows)
	}
	if len(slack.tokens) != 1 || slack.tokens[0] != "xoxp-user" {
		t.Fatalf("tokens %#v", slack.tokens)
	}
}

func TestScreenConversationsSortUnreadFirst(t *testing.T) {
	slack := &fakeSlack{
		t: t,
		conversations: map[string][]map[string]any{
			"im,mpim":                        {{"id": "D1", "is_im": true, "user": "U9"}},
			"public_channel,private_channel": {{"id": "C1", "name": "general", "is_channel": true, "unread_count_display": float64(4)}},
		},
		users: map[string]map[string]any{"U9": {"id": "U9", "profile": map[string]any{"display_name": "Ada"}}},
	}
	client := Session{Token: "xoxp-user", Gateway: slack}
	rows, err := ScreenConversations(client)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Name != "general" || rows[0].Unread != 4 || rows[1].Name != "Ada" || rows[1].Surface != "dm" {
		t.Fatalf("%#v", rows)
	}
}
