package cachedb

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/slack/internal/app"
)

const eventCacheV1 = `
CREATE TABLE conversations (
  channel_id TEXT PRIMARY KEY, surface TEXT NOT NULL, conversation TEXT NOT NULL, name TEXT NOT NULL,
  email TEXT NOT NULL, members TEXT NOT NULL, user_id TEXT NOT NULL, last_read TEXT NOT NULL,
  info_json TEXT NOT NULL, latest_ts REAL NOT NULL DEFAULT 0, unread_ts REAL NOT NULL DEFAULT 0,
  history_loaded INTEGER NOT NULL DEFAULT 0, updated_at TEXT NOT NULL);
CREATE TABLE messages (
  message_id TEXT PRIMARY KEY, channel_id TEXT NOT NULL, ts TEXT NOT NULL, sort_ts REAL NOT NULL,
  user_id TEXT NOT NULL, text TEXT NOT NULL, unread INTEGER NOT NULL DEFAULT 0, sender_json TEXT NOT NULL,
  message_json TEXT NOT NULL, event_id TEXT, updated_at TEXT NOT NULL);
CREATE INDEX idx_messages_channel_ts ON messages(channel_id, sort_ts DESC);
CREATE TABLE events (event_id TEXT PRIMARY KEY, channel_id TEXT NOT NULL, ts TEXT NOT NULL, received_at TEXT NOT NULL);
INSERT INTO events VALUES('E1', 'D1', '100.000100', 'now'), ('E2', 'D9', '1.000000', 'now');
CREATE TABLE cache_state (key TEXT PRIMARY KEY, value TEXT NOT NULL);
INSERT INTO cache_state VALUES('schema_version', '1'), ('last_sync_at', '2026-11-16T15:47:00+05:30'), ('last_sync_conversations', '3'), ('last_sync_messages', '9');
INSERT INTO conversations VALUES('D1', 'dm', 'x', 'x', '-', '-', 'U2', '150.000000', '{"last_read":"150.000000","is_im":true}', 200, 200, 1, '2026-11-16T15:47:00+05:30');
INSERT INTO messages VALUES('D1:100.000100', 'D1', '100.000100', 100.0001, 'U2', 'old', 0, '{"id":"U2"}', '{"ts":"100.000100","user":"U2","text":"old"}', 'E1', 'now');
INSERT INTO messages VALUES('D1:200.000100', 'D1', '200.000100', 200.0001, 'U2', 'new', 1, '{"id":"U2"}', '{"user":"U2","text":"new"}', NULL, 'now');`

func TestEventCacheMigratesVersionOne(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(eventCacheV1); err != nil {
		t.Fatal(err)
	}
	raw.Close()
	db, err := eventCacheConnect(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if got := eventCacheVersion(db.QueryRow); got != 2 {
		t.Fatalf("version %d", got)
	}
	var at, conversations, messages int64
	if err := db.QueryRow("SELECT last_sync_at, conversations, messages FROM sync_state").Scan(&at, &conversations, &messages); err != nil || at != 1794824220 || conversations != 3 || messages != 9 {
		t.Fatalf("sync state %d %d %d %v", at, conversations, messages, err)
	}
	var updated int64
	if err := db.QueryRow("SELECT updated_at FROM messages LIMIT 1").Scan(&updated); err != nil {
		t.Fatalf("updated_at is not a number: %v", err)
	}
	var info, lastRead string
	if err := db.QueryRow("SELECT info_json, last_read FROM conversations WHERE channel_id = 'D1'").Scan(&info, &lastRead); err != nil {
		t.Fatal(err)
	}
	if info != `{"is_im":true}` || lastRead != "150.000000" {
		t.Fatalf("info %s last_read %s", info, lastRead)
	}
	for _, column := range []string{"message_id", "user_id", "text", "event_id"} {
		if _, err := db.Exec("SELECT " + column + " FROM messages"); err == nil {
			t.Fatalf("column %s still stored", column)
		}
	}
	entries, err := eventCacheLoadEntries(path, "U1", 10, "D1")
	if err != nil || len(entries) != 2 {
		t.Fatalf("entries %+v err %v", entries, err)
	}
	if str(entries[1].Message["ts"]) != "200.000100" || !entries[1].Unread || entries[0].Unread {
		t.Fatalf("entries %+v", entries)
	}
	var orphan string
	if err := db.QueryRow("SELECT surface FROM conversations WHERE channel_id = 'D9'").Scan(&orphan); err != nil || orphan != "dm" {
		t.Fatalf("placeholder conversation %q %v", orphan, err)
	}
	if _, err := db.Exec("DELETE FROM conversations WHERE channel_id = 'D1'"); err != nil {
		t.Fatal(err)
	}
	var left int
	if err := db.QueryRow("SELECT (SELECT count(*) FROM messages WHERE channel_id = 'D1') + (SELECT count(*) FROM events WHERE channel_id = 'D1')").Scan(&left); err != nil || left != 0 {
		t.Fatalf("rows left after the conversation went %d %v", left, err)
	}
	if _, err := db.Exec("INSERT INTO messages(channel_id, message_json, sender_json, updated_at) VALUES('NOPE', '{\"ts\":\"1\"}', '{}', 'now')"); err == nil {
		t.Fatal("a message without a conversation was stored")
	}
	if _, err := eventCacheConnect(path); err != nil {
		t.Fatal(err)
	}
}

func TestEventCacheStoresLastReadOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.db")
	row := app.ConversationRow{ChannelID: "D1", Surface: "dm", LastRead: "5.000000", Info: map[string]any{"last_read": "9.000000", "is_im": true}}
	if _, err := eventCacheStoreConversationRow(path, row, false); err != nil {
		t.Fatal(err)
	}
	if err := eventCacheMarkRead(path, "D1", "12.000000"); err != nil {
		t.Fatal(err)
	}
	db, err := eventCacheConnect(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var info, lastRead string
	if err := db.QueryRow("SELECT info_json, last_read FROM conversations WHERE channel_id = 'D1'").Scan(&info, &lastRead); err != nil {
		t.Fatal(err)
	}
	if info != `{"is_im":true}` || lastRead != "12.000000" {
		t.Fatalf("info %s last_read %s", info, lastRead)
	}
}

func TestLiveConversationUpsertClearsStickyUnread(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "events.db")
	if _, err := eventCacheStoreEntries(cachePath, []app.MessageEntry{{
		SortTS:       100.0001,
		Email:        "-",
		ChannelID:    "D1",
		Surface:      "dm",
		Conversation: "x",
		UserID:       "U2",
		Members:      "-",
		Message:      map[string]any{"ts": "100.000100", "user": "U2", "text": "old"},
		Sender:       map[string]any{"id": "U2"},
		Unread:       true,
	}}, "", true); err != nil {
		t.Fatal(err)
	}
	if _, err := eventCacheStoreConversationRow(cachePath, app.ConversationRow{
		ChannelID: "D1",
		Surface:   "dm",
		Unread:    0,
		LastRead:  "200.000100",
		LatestTS:  "200.000100",
	}, true); err != nil {
		t.Fatal(err)
	}
	db, err := eventCacheConnect(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var unreadTS float64
	if err := db.QueryRow("SELECT unread_ts FROM conversations WHERE channel_id = ?", "D1").Scan(&unreadTS); err != nil {
		t.Fatal(err)
	}
	if unreadTS != 0 {
		t.Fatalf("expected unread_ts 0 after live read upsert, got %v", unreadTS)
	}
}
