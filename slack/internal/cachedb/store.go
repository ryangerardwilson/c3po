package cachedb

import (
	"database/sql"
	"os"

	"github.com/ryangerardwilson/c3po/slack/internal/anyval"
	"github.com/ryangerardwilson/c3po/slack/internal/app"
)

// Store is the event cache. Each call opens the named file and closes it.
type Store struct{}

var _ app.Cache = Store{}

// StoreConversation writes a conversation and its messages in one transaction.
func (Store) StoreConversation(path string, row app.ConversationRow, historyLoaded bool) (int, error) {
	return eventCacheStoreConversationRow(path, row, historyLoaded)
}

// StoreEntries writes messages in one transaction. A non-empty eventID is
// claimed once; a message whose event was already claimed is skipped.
func (Store) StoreEntries(path string, entries []app.MessageEntry, eventID string, historyLoaded bool) (int, error) {
	return eventCacheStoreEntries(path, entries, eventID, historyLoaded)
}

// MarkRead moves a cached conversation's last_read to latestTS.
func (Store) MarkRead(path, channelID, latestTS string) error {
	return eventCacheMarkRead(path, channelID, latestTS)
}

// LoadEntries reads the newest cached messages, oldest first.
func (Store) LoadEntries(path, selfUserID string, limit int, channelID string) ([]app.MessageEntry, error) {
	return eventCacheLoadEntries(path, selfUserID, limit, channelID)
}

// HistoryLoaded says which conversations had their history loaded.
func (Store) HistoryLoaded(path string) map[string]bool {
	return eventCacheHistoryLoadedMap(path)
}

// RecordSync replaces the one sync_state row.
func (Store) RecordSync(path string, conversations, messages int) error {
	db, err := eventCacheConnect(path)
	if err != nil {
		return err
	}
	defer db.Close()
	return eventCacheRecordSync(db, conversations, messages)
}

// Status counts conversations and messages and reads the last sync.
func (Store) Status(path string) (app.CacheStatus, error) {
	db, err := eventCacheConnect(path)
	if err != nil {
		return app.CacheStatus{}, err
	}
	defer db.Close()
	status := app.CacheStatus{
		Conversations: countRows(db, "conversations"),
		Messages:      countRows(db, "messages"),
	}
	status.LastSyncAt, status.LastSyncConversations, status.LastSyncMessages = eventCacheLastSync(db)
	return status, nil
}

// Reset removes the cache file and its WAL and shared-memory files.
func (Store) Reset(path string) {
	for _, file := range []string{path, path + "-wal", path + "-shm"} {
		_ = os.Remove(file)
	}
}

func countRows(db *sql.DB, table string) int {
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
	return count
}

func str(value any) string {
	return anyval.Str(value)
}

func tsFloat(value string) float64 {
	return anyval.TSFloat(value)
}

func firstNonEmpty(values ...string) string {
	return anyval.FirstNonEmpty(values...)
}

func maxFloat(values ...float64) float64 {
	return anyval.MaxFloat(values...)
}

func maxInt(a, b int) int {
	return anyval.MaxInt(a, b)
}
