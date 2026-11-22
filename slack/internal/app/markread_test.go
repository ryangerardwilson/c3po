package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestMarkAllReadMarksLiveUnreadAndCache(t *testing.T) {
	slack := &fakeSlack{
		t: t,
		conversations: map[string][]map[string]any{"im,mpim,public_channel,private_channel": {
			{"id": "D1", "is_im": true, "unread_count_display": float64(1), "unread_count": float64(1), "latest": map[string]any{"ts": "100.000100"}},
			{"id": "G1", "is_mpim": true, "unread_count_display": float64(1), "unread_count": float64(1), "latest": map[string]any{"ts": "101.000100"}},
			{"id": "C7", "name": "quiet", "unread_count_display": float64(0)},
		}},
	}
	cache := &fakeCache{}
	var stdout bytes.Buffer
	rt := &Runtime{Stdout: &stdout, Stderr: &bytes.Buffer{}, Slack: slack, Cache: cache}
	result, err := rt.markAllUnreadNotificationsAsRead(rt.slackClient("xoxp-token"), "events.db", "1")
	if err != nil {
		t.Fatalf("mark all read: %v", err)
	}
	if result.Marked != 2 || result.Failed != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if strings.Join(slack.marks, ",") != "G1:101.000100,D1:100.000100" {
		t.Fatalf("marks newest first, live unread only: %#v", slack.marks)
	}
	if strings.Join(cache.marks, ",") != "events.db|G1:101.000100,events.db|D1:100.000100" {
		t.Fatalf("cache marks: %#v", cache.marks)
	}
	if !strings.Contains(stdout.String(), "Summary: marked_read=2 failed=0") {
		t.Fatalf("missing summary: %s", stdout.String())
	}
}

func TestMarkAllReadNamesTheMissingScope(t *testing.T) {
	slack := &fakeSlack{
		t: t,
		conversations: map[string][]map[string]any{"im,mpim,public_channel,private_channel": {
			{"id": "G1", "is_mpim": true, "unread_count": float64(2), "latest": map[string]any{"ts": "101.000100"}},
		}},
		markErrs: map[string]error{"G1": &APIError{Method: "conversations.mark", Code: "missing_scope"}},
	}
	cache := &fakeCache{}
	var stdout bytes.Buffer
	rt := &Runtime{Stdout: &stdout, Stderr: &bytes.Buffer{}, Slack: slack, Cache: cache}
	result, err := rt.markAllUnreadNotificationsAsRead(rt.slackClient("xoxp-token"), "events.db", "")
	if err != nil {
		t.Fatalf("mark all read: %v", err)
	}
	if result.Marked != 0 || result.Failed != 1 || len(cache.marks) != 0 {
		t.Fatalf("result %+v cache %#v", result, cache.marks)
	}
	if !strings.Contains(stdout.String(), "action: failed:missing_scope:add mpim:write to user token") {
		t.Fatalf("stdout: %s", stdout.String())
	}
}

func TestUnreadTotalsCountLiveConversations(t *testing.T) {
	slack := &fakeSlack{
		t: t,
		conversations: map[string][]map[string]any{"im,mpim,public_channel,private_channel": {
			{"id": "C1", "name": "secret-channel", "unread_count_display": float64(3), "unread_count": float64(3)},
			{"id": "D2", "is_im": true, "unread_count_display": float64(0), "unread_count": float64(0), "has_unreads": true},
		}},
	}
	count, conversations, err := unreadNotificationTotals(slack)
	if err != nil || count != 4 || conversations != 2 {
		t.Fatalf("count %d conversations %d err %v", count, conversations, err)
	}
}
