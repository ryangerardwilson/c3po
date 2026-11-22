// Package cachedb is the sqlite event cache behind the core's Cache port.
// It owns the schema, its migration, and every query.
package cachedb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/ryangerardwilson/c3po/slack/internal/app"

	_ "modernc.org/sqlite"
)

// schemaVersion is the PRAGMA user_version this cache writes.
const schemaVersion = 2

func jsonText(value any) string {
	if value == nil {
		value = map[string]any{}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func jsonMap(value string) map[string]any {
	out := map[string]any{}
	_ = json.Unmarshal([]byte(value), &out)
	return out
}

func eventCacheConnect(path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("empty event cache path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;`); err != nil {
		db.Close()
		return nil, err
	}
	if err := eventCacheInit(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// The event cache is a copy of Slack's answers and is never the source of
// truth. A history load or events sync refreshes it, a mark-read writes it
// only after Slack accepts the mark, and events reset cache drops the file.
// history_loaded and sync_state are local facts about this copy. Each fact
// is stored once.
// last_read lives only in its conversations column, never in info_json.
// A message's ts lives only in message_json; ts and sort_ts are generated
// from it and are not stored. The events table is the one record of which
// event delivered which message. Times are unix seconds in UTC. sync_state is
// the one row describing the last sync, and the schema version is
// PRAGMA user_version. A message and an event belong to their
// conversation and go with it; an event is claimed before its conversation
// is written in the same transaction, so that key is checked at commit.
const eventCacheTables = `
CREATE TABLE IF NOT EXISTS conversations (
  channel_id TEXT PRIMARY KEY,
  surface TEXT NOT NULL CHECK (surface IN ('dm', 'group_dm', 'private_channel', 'channel')),
  conversation TEXT NOT NULL,
  name TEXT NOT NULL,
  email TEXT NOT NULL,
  members TEXT NOT NULL,
  user_id TEXT NOT NULL,
  last_read TEXT NOT NULL,
  info_json TEXT NOT NULL CHECK (json_valid(info_json)),
  latest_ts REAL NOT NULL DEFAULT 0,
  unread_ts REAL NOT NULL DEFAULT 0,
  history_loaded INTEGER NOT NULL DEFAULT 0 CHECK (history_loaded IN (0, 1)),
  updated_at INTEGER NOT NULL,
  CHECK (unread_ts = 0 OR unread_ts > CAST(last_read AS REAL))
) STRICT;
CREATE INDEX IF NOT EXISTS idx_conversations_unread ON conversations(unread_ts DESC) WHERE unread_ts > 0;
CREATE TABLE IF NOT EXISTS events (
  event_id TEXT PRIMARY KEY,
  channel_id TEXT NOT NULL REFERENCES conversations(channel_id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
  ts TEXT NOT NULL,
  received_at INTEGER NOT NULL
) STRICT;
CREATE INDEX IF NOT EXISTS idx_events_channel ON events(channel_id);
CREATE TABLE IF NOT EXISTS sync_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  last_sync_at INTEGER NOT NULL,
  conversations INTEGER NOT NULL CHECK (conversations >= 0),
  messages INTEGER NOT NULL CHECK (messages >= 0)
) STRICT;`

const eventCacheMessages = `
CREATE TABLE IF NOT EXISTS messages (
  id INTEGER PRIMARY KEY,
  channel_id TEXT NOT NULL REFERENCES conversations(channel_id) ON DELETE CASCADE,
  message_json TEXT NOT NULL,
  ts TEXT GENERATED ALWAYS AS (json_extract(message_json, '$.ts')) VIRTUAL,
  sort_ts REAL GENERATED ALWAYS AS (CAST(json_extract(message_json, '$.ts') AS REAL)) VIRTUAL,
  sender_json TEXT NOT NULL,
  updated_at INTEGER NOT NULL
) STRICT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_channel_key ON messages(channel_id, ts);
CREATE INDEX IF NOT EXISTS idx_messages_channel_ts ON messages(channel_id, sort_ts DESC);
CREATE INDEX IF NOT EXISTS idx_messages_sort_ts ON messages(sort_ts DESC);`

// eventCacheV2 moves a version 1 cache to version 2. It keeps every message,
// event, and conversation; a message or event whose conversation is missing
// gets a placeholder conversation, an unknown surface becomes dm, and a flag
// other than 0 becomes 1. An unread mark at or before last_read is cleared.
// A message's unread flag is dropped; it is derived from last_read. A message whose payload lacks ts takes it
// from the old ts column. The old message_id, ts, sort_ts, user_id, text, and
// event_id columns are dropped, last_read leaves info_json, text times
// become unix seconds, and cache_state becomes the one sync_state row.
const eventCacheV2 = `
CREATE TABLE conversations_v2 (
  channel_id TEXT PRIMARY KEY,
  surface TEXT NOT NULL CHECK (surface IN ('dm', 'group_dm', 'private_channel', 'channel')),
  conversation TEXT NOT NULL,
  name TEXT NOT NULL,
  email TEXT NOT NULL,
  members TEXT NOT NULL,
  user_id TEXT NOT NULL,
  last_read TEXT NOT NULL,
  info_json TEXT NOT NULL CHECK (json_valid(info_json)),
  latest_ts REAL NOT NULL DEFAULT 0,
  unread_ts REAL NOT NULL DEFAULT 0,
  history_loaded INTEGER NOT NULL DEFAULT 0 CHECK (history_loaded IN (0, 1)),
  updated_at INTEGER NOT NULL,
  CHECK (unread_ts = 0 OR unread_ts > CAST(last_read AS REAL))
) STRICT;
INSERT INTO conversations_v2 SELECT channel_id,
  CASE WHEN surface IN ('dm', 'group_dm', 'private_channel', 'channel') THEN surface ELSE 'dm' END,
  conversation, name, email, members, user_id, last_read,
  CASE WHEN json_valid(info_json) THEN json_remove(info_json, '$.last_read') ELSE '{}' END,
  latest_ts, CASE WHEN unread_ts > CAST(last_read AS REAL) THEN unread_ts ELSE 0 END,
  CASE WHEN history_loaded = 0 THEN 0 ELSE 1 END, coalesce(unixepoch(updated_at), 0)
FROM conversations;
DROP TABLE conversations;
ALTER TABLE conversations_v2 RENAME TO conversations;
INSERT OR IGNORE INTO conversations(channel_id, surface, conversation, name, email, members, user_id, last_read, info_json, updated_at)
SELECT DISTINCT channel_id, 'dm', channel_id, channel_id, '-', '-', '-', '0', '{}', 0
FROM (SELECT channel_id FROM messages UNION SELECT channel_id FROM events)
WHERE channel_id NOT IN (SELECT channel_id FROM conversations);
CREATE TABLE events_v2 (
  event_id TEXT PRIMARY KEY,
  channel_id TEXT NOT NULL REFERENCES conversations(channel_id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
  ts TEXT NOT NULL,
  received_at INTEGER NOT NULL
) STRICT;
INSERT INTO events_v2 SELECT event_id, channel_id, ts, coalesce(unixepoch(received_at), 0) FROM events;
DROP TABLE events;
ALTER TABLE events_v2 RENAME TO events;
CREATE INDEX IF NOT EXISTS idx_events_channel ON events(channel_id);
CREATE TABLE messages_v2 (
  id INTEGER PRIMARY KEY,
  channel_id TEXT NOT NULL REFERENCES conversations(channel_id) ON DELETE CASCADE,
  message_json TEXT NOT NULL,
  ts TEXT GENERATED ALWAYS AS (json_extract(message_json, '$.ts')) VIRTUAL,
  sort_ts REAL GENERATED ALWAYS AS (CAST(json_extract(message_json, '$.ts') AS REAL)) VIRTUAL,
  sender_json TEXT NOT NULL,
  updated_at INTEGER NOT NULL
) STRICT;
INSERT INTO messages_v2(channel_id, message_json, sender_json, updated_at)
SELECT channel_id,
       CASE WHEN json_valid(message_json) AND json_type(message_json, '$.ts') = 'text' THEN message_json
            ELSE json_set(CASE WHEN json_valid(message_json) THEN message_json ELSE '{}' END, '$.ts', ts) END,
       sender_json, coalesce(unixepoch(updated_at), 0)
FROM messages ORDER BY sort_ts;
DROP TABLE messages;
ALTER TABLE messages_v2 RENAME TO messages;
CREATE TABLE sync_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  last_sync_at INTEGER NOT NULL,
  conversations INTEGER NOT NULL CHECK (conversations >= 0),
  messages INTEGER NOT NULL CHECK (messages >= 0)
) STRICT;
INSERT INTO sync_state(id, last_sync_at, conversations, messages)
SELECT 1, coalesce(unixepoch(value), 0),
       max(0, coalesce((SELECT CAST(value AS INTEGER) FROM cache_state WHERE key = 'last_sync_conversations'), 0)),
       max(0, coalesce((SELECT CAST(value AS INTEGER) FROM cache_state WHERE key = 'last_sync_messages'), 0))
FROM cache_state WHERE key = 'last_sync_at';
DROP TABLE cache_state;
PRAGMA user_version = 2;`

func eventCacheInit(db *sql.DB) error {
	if eventCacheVersion(db.QueryRow) == 1 {
		if err := eventCacheMigrate(db, eventCacheV2); err != nil {
			return err
		}
	}
	if _, err := db.Exec(eventCacheTables + eventCacheMessages); err != nil {
		return err
	}
	_, err := db.Exec("PRAGMA user_version = " + strconv.Itoa(schemaVersion))
	return err
}

// eventCacheVersion is PRAGMA user_version. A version 1 cache kept its
// version in cache_state instead, and a new file has neither.
func eventCacheVersion(queryRow func(string, ...any) *sql.Row) int {
	var version int
	_ = queryRow("PRAGMA user_version").Scan(&version)
	if version > 0 {
		return version
	}
	var old string
	if queryRow("SELECT value FROM cache_state WHERE key = 'schema_version'").Scan(&old) == nil {
		n, _ := strconv.Atoi(old)
		return n
	}
	return 0
}

// eventCacheMigrate runs body once in one immediate transaction when the
// cache is still version 1. body sets the new version. A second process that
// waited sees it and runs nothing. Foreign keys are off while tables are
// rebuilt so dropping an old parent table does not delete its children; a
// key the rebuild broke fails the migration.
func eventCacheMigrate(db *sql.DB, body string) error {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, "PRAGMA foreign_keys = ON")
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	connRow := func(query string, args ...any) *sql.Row { return conn.QueryRowContext(ctx, query, args...) }
	if eventCacheVersion(connRow) != 1 {
		_, err := conn.ExecContext(ctx, "ROLLBACK")
		return err
	}
	if _, err := conn.ExecContext(ctx, body); err != nil {
		_, _ = conn.ExecContext(ctx, "ROLLBACK")
		return err
	}
	var broken int
	if err := conn.QueryRowContext(ctx, "SELECT count(*) FROM pragma_foreign_key_check").Scan(&broken); err != nil || broken > 0 {
		_, _ = conn.ExecContext(ctx, "ROLLBACK")
		return fmt.Errorf("event cache migration left %d broken keys: %v", broken, err)
	}
	_, err = conn.ExecContext(ctx, "COMMIT")
	return err
}

// eventCacheNow is the current time in unix seconds, UTC.
func eventCacheNow() int64 {
	return time.Now().Unix()
}

// eventCacheRecordSync replaces the one sync_state row.
func eventCacheRecordSync(db *sql.DB, conversations, messages int) error {
	_, err := db.Exec("INSERT INTO sync_state(id, last_sync_at, conversations, messages) VALUES(1, ?, ?, ?) "+
		"ON CONFLICT(id) DO UPDATE SET last_sync_at = excluded.last_sync_at, conversations = excluded.conversations, messages = excluded.messages",
		eventCacheNow(), conversations, messages)
	return err
}

// eventCacheLastSync returns the last sync time in local RFC3339 and its
// counts, or an empty time and zero counts before the first sync.
func eventCacheLastSync(db *sql.DB) (string, int, int) {
	var at int64
	var conversations, messages int
	if db.QueryRow("SELECT last_sync_at, conversations, messages FROM sync_state WHERE id = 1").Scan(&at, &conversations, &messages) != nil {
		return "", 0, 0
	}
	return time.Unix(at, 0).Local().Format(time.RFC3339), conversations, messages
}

// cacheStore is a database or a transaction. Each cache write takes one, so
// a conversation and a message have one write path each.
type cacheStore interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

func eventCacheClaimEvent(db cacheStore, eventID, channelID, ts string) bool {
	if eventID == "" {
		return true
	}
	_, err := db.Exec("INSERT INTO events(event_id, channel_id, ts, received_at) VALUES(?, ?, ?, ?)", eventID, channelID, ts, eventCacheNow())
	return err == nil
}

func conversationInfoFromEntry(entry app.MessageEntry) app.ConversationRow {
	return app.ConversationRow{
		ChannelID:    entry.ChannelID,
		Surface:      firstNonEmpty(entry.Surface, "dm"),
		Conversation: firstNonEmpty(entry.Conversation, entry.Email, entry.ChannelID, "-"),
		Name:         firstNonEmpty(entry.Conversation, entry.Email, entry.ChannelID, "-"),
		Email:        firstNonEmpty(entry.Email, "-"),
		Members:      firstNonEmpty(entry.Members, "-"),
		UserID:       firstNonEmpty(entry.UserID, str(entry.Sender["id"]), "-"),
		LastRead:     "0",
		Info:         map[string]any{},
	}
}

// eventCacheUpsertConversation is the one write path for a conversation row.
// A last_read from Slack's info folds into the last_read column and is not
// kept in info_json.
func eventCacheUpsertConversation(db cacheStore, row app.ConversationRow, latestTS float64, unreadTS float64, historyLoaded bool) (float64, error) {
	if row.ChannelID == "" {
		return 0, nil
	}
	var existingInfoJSON, existingLastRead string
	var existingLatest, existingUnread float64
	var existingHistory int
	err := db.QueryRow("SELECT info_json, last_read, latest_ts, unread_ts, history_loaded FROM conversations WHERE channel_id = ?", row.ChannelID).Scan(&existingInfoJSON, &existingLastRead, &existingLatest, &existingUnread, &existingHistory)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	info := jsonMap(existingInfoJSON)
	for key, value := range row.Info {
		if str(value) != "" && str(value) != "-" {
			info[key] = value
		}
	}
	lastRead := maxFloat(tsFloat(str(info["last_read"])), tsFloat(existingLastRead), tsFloat(row.LastRead))
	delete(info, "last_read")
	if lastRead > 0 {
		row.LastRead = fmt.Sprintf("%.6f", lastRead)
	} else if row.LastRead == "" {
		row.LastRead = "0"
	}
	if latestTS < existingLatest {
		latestTS = existingLatest
	}
	if existingHistory == 1 {
		historyLoaded = true
	}
	if unreadTS > 0 && unreadTS <= lastRead {
		unreadTS = 0
	}
	_, err = db.Exec(`
INSERT INTO conversations(
  channel_id, surface, conversation, name, email, members, user_id,
  last_read, info_json, latest_ts, unread_ts, history_loaded, updated_at
) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(channel_id) DO UPDATE SET
  surface = excluded.surface, conversation = excluded.conversation, name = excluded.name,
  email = excluded.email, members = excluded.members, user_id = excluded.user_id,
  last_read = excluded.last_read, info_json = excluded.info_json, latest_ts = excluded.latest_ts,
  unread_ts = excluded.unread_ts, history_loaded = excluded.history_loaded, updated_at = excluded.updated_at`,
		row.ChannelID,
		firstNonEmpty(row.Surface, "dm"),
		firstNonEmpty(row.Conversation, row.Name, row.ChannelID),
		firstNonEmpty(row.Name, row.Conversation, row.ChannelID),
		firstNonEmpty(row.Email, "-"),
		firstNonEmpty(row.Members, "-"),
		firstNonEmpty(row.UserID, "-"),
		row.LastRead,
		jsonText(info),
		latestTS,
		unreadTS,
		boolToInt(historyLoaded),
		eventCacheNow(),
	)
	return lastRead, err
}

// messagePayload is the Slack message with ts as a string, which is the one
// place the cache keeps a message's ts.
func messagePayload(message map[string]any, ts string) map[string]any {
	if value, ok := message["ts"].(string); ok && value == ts {
		return message
	}
	out := make(map[string]any, len(message)+1)
	for key, value := range message {
		out[key] = value
	}
	out["ts"] = ts
	return out
}

// eventCacheUpsertEntry is the one write path for a message row.
func eventCacheUpsertEntry(db cacheStore, entry app.MessageEntry, eventID string, historyLoaded bool) (bool, error) {
	channelID := entry.ChannelID
	ts := str(entry.Message["ts"])
	if channelID == "" || ts == "" {
		return false, nil
	}
	sortTS := entry.SortTS
	if sortTS == 0 {
		sortTS = tsFloat(ts)
	}
	if !eventCacheClaimEvent(db, eventID, channelID, ts) {
		return false, nil
	}
	unreadTS := 0.0
	if entry.Unread {
		unreadTS = sortTS
	}
	row := conversationInfoFromEntry(entry)
	if _, err := eventCacheUpsertConversation(db, row, sortTS, unreadTS, historyLoaded); err != nil {
		return false, err
	}
	_, err := db.Exec(`
INSERT INTO messages(channel_id, message_json, sender_json, updated_at)
VALUES(?, ?, ?, ?)
ON CONFLICT(channel_id, ts) DO UPDATE SET
  message_json = excluded.message_json, sender_json = excluded.sender_json, updated_at = excluded.updated_at`,
		channelID,
		jsonText(messagePayload(entry.Message, ts)),
		jsonText(entry.Sender),
		eventCacheNow(),
	)
	return err == nil, err
}

func eventCacheStoreEntries(cachePath string, entries []app.MessageEntry, eventID string, historyLoaded bool) (int, error) {
	if cachePath == "" {
		return 0, nil
	}
	db, err := eventCacheConnect(cachePath)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		ok, err := eventCacheUpsertEntry(tx, entry, eventID, historyLoaded)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
		if ok {
			count++
		}
	}
	return count, tx.Commit()
}

func eventCacheStoreConversationRow(cachePath string, row app.ConversationRow, historyLoaded bool) (int, error) {
	if cachePath == "" || row.ChannelID == "" {
		return 0, nil
	}
	db, err := eventCacheConnect(cachePath)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	latest := tsFloat(row.LatestTS)
	unreadTS := 0.0
	if row.Unread > 0 {
		unreadTS = latest
	}
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	if _, err := eventCacheUpsertConversation(tx, row, latest, unreadTS, historyLoaded || row.HistoryLoaded); err != nil {
		tx.Rollback()
		return 0, err
	}
	count := 0
	for _, entry := range row.Messages {
		if entry.ChannelID == "" {
			entry.ChannelID = row.ChannelID
		}
		ok, err := eventCacheUpsertEntry(tx, entry, "", historyLoaded)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
		if ok {
			count++
		}
	}
	return count, tx.Commit()
}

func eventCacheLoadEntries(cachePath, selfUserID string, limit int, channelID string) ([]app.MessageEntry, error) {
	if cachePath == "" {
		return nil, nil
	}
	if _, err := os.Stat(cachePath); err != nil {
		return nil, nil
	}
	db, err := eventCacheConnect(cachePath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	query := `
SELECT messages.channel_id, messages.ts, messages.sort_ts,
       messages.sender_json, messages.message_json, conversations.info_json,
       conversations.surface, conversations.conversation, conversations.name,
       conversations.email, conversations.members, conversations.user_id,
       conversations.last_read
FROM messages
JOIN conversations ON conversations.channel_id = messages.channel_id`
	var args []any
	if channelID != "" {
		query += " WHERE messages.channel_id = ?"
		args = append(args, channelID)
	}
	query += " ORDER BY messages.sort_ts DESC LIMIT ?"
	args = append(args, maxInt(1, limit))
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []app.MessageEntry
	for rows.Next() {
		var channelID, ts, senderJSON, messageJSON, infoJSON, surface, conversation, name, email, members, userID, lastRead string
		var sortTS float64
		if err := rows.Scan(&channelID, &ts, &sortTS, &senderJSON, &messageJSON, &infoJSON, &surface, &conversation, &name, &email, &members, &userID, &lastRead); err != nil {
			return nil, err
		}
		message := jsonMap(messageJSON)
		sender := jsonMap(senderJSON)
		// A message is unread when it is newer than last_read and not the
		// operator's own.
		unread := sortTS > tsFloat(lastRead) && (selfUserID == "" || str(message["user"]) != selfUserID)
		_ = infoJSON
		entries = append(entries, app.MessageEntry{
			SortTS:       sortTS,
			Email:        firstNonEmpty(email, "-"),
			ChannelID:    channelID,
			Surface:      firstNonEmpty(surface, "dm"),
			Conversation: firstNonEmpty(conversation, name, channelID),
			UserID:       firstNonEmpty(userID, "-"),
			Members:      firstNonEmpty(members, "-"),
			Message:      message,
			Sender:       sender,
			Unread:       unread,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].SortTS < entries[j].SortTS })
	return entries, rows.Err()
}

func eventCacheHistoryLoadedMap(cachePath string) map[string]bool {
	out := map[string]bool{}
	if _, err := os.Stat(cachePath); err != nil {
		return out
	}
	db, err := eventCacheConnect(cachePath)
	if err != nil {
		return out
	}
	defer db.Close()
	rows, err := db.Query("SELECT channel_id, history_loaded FROM conversations")
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var channelID string
		var loaded int
		if rows.Scan(&channelID, &loaded) == nil {
			out[channelID] = loaded == 1
		}
	}
	return out
}

// eventCacheMarkRead moves last_read to latestTS through the one
// conversation write path, which also clears the unread mark.
func eventCacheMarkRead(cachePath, channelID, latestTS string) error {
	if cachePath == "" || channelID == "" || latestTS == "" {
		return nil
	}
	db, err := eventCacheConnect(cachePath)
	if err != nil {
		return err
	}
	defer db.Close()
	var row app.ConversationRow
	var latest float64
	var history int
	err = db.QueryRow("SELECT surface, conversation, name, email, members, user_id, latest_ts, history_loaded FROM conversations WHERE channel_id = ?", channelID).
		Scan(&row.Surface, &row.Conversation, &row.Name, &row.Email, &row.Members, &row.UserID, &latest, &history)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	row.ChannelID = channelID
	row.LastRead = latestTS
	_, err = eventCacheUpsertConversation(db, row, latest, 0, history == 1)
	return err
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
