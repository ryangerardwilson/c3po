// Package syncstate keeps Drive sync state for the syncer: the table
// gdrive_sync_entry for a production record path, and a JSON file otherwise.
package syncstate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ryangerardwilson/c3po/gdrive/internal/paths"
	"github.com/ryangerardwilson/c3po/gdrive/internal/syncer"
	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

// Store is the syncer.StateStore the gdrive commands use.
type Store struct{}

type statePayload struct {
	Entries []syncer.StateEntry `json:"entries"`
}

// StateFile is <data>/state/<preset>-<id>.json. For preset 1 a legacy
// <data>/state/<id>.json is renamed into place.
func StateFile(preset, regID string) string {
	_ = paths.EnsureDirs()
	path := filepath.Join(paths.StateDir(), preset+"-"+regID+".json")
	legacy := filepath.Join(paths.StateDir(), regID+".json")
	if preset == "1" && !exists(path) && exists(legacy) {
		_ = os.Rename(legacy, path)
	}
	return path
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Load reads one registration's sync entries. A production record path
// reads gdrive_sync_entry; another path is a JSON file.
func (Store) Load(preset, regID string) (map[string]syncer.StateEntry, error) {
	path := StateFile(preset, regID)
	var entries []syncer.StateEntry
	if sqlxdb.UsesDB(path) {
		var rows []syncer.StateEntry
		sql := "SELECT relpath, kind, drive_id, size, mtime_ns, sha1 FROM gdrive_sync_entry WHERE " + stateWhere(preset, regID)
		if err := sqlxdb.Query(sql, &rows); err != nil {
			return nil, err
		}
		entries = rows
	} else {
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			return map[string]syncer.StateEntry{}, nil
		}
		if err != nil {
			return nil, err
		}
		payload := statePayload{}
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, err
		}
		entries = payload.Entries
	}
	result := map[string]syncer.StateEntry{}
	for _, entry := range entries {
		entry.ParentRelPath = syncer.ParentRelPath(entry.RelPath)
		result[entry.RelPath] = entry
	}
	return result, nil
}

// Save makes one registration's stored entries match entries in one
// transaction. An entry that stays keeps its row.
func (Store) Save(preset, regID string, entries map[string]syncer.StateEntry) error {
	path := StateFile(preset, regID)
	keys := sortedKeys(entries)
	if sqlxdb.UsesDB(path) {
		return sqlxdb.Exec(saveStateSQL(preset, regID, keys, entries))
	}
	payload := statePayload{Entries: make([]syncer.StateEntry, 0, len(keys))}
	for _, key := range keys {
		payload.Entries = append(payload.Entries, entries[key])
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func saveStateSQL(preset, regID string, keys []string, entries map[string]syncer.StateEntry) string {
	kept := []string{"''"}
	var b strings.Builder
	for _, key := range keys {
		e := entries[key]
		kept = append(kept, sqlxdb.Quote(e.RelPath))
		b.WriteString(";\nINSERT INTO gdrive_sync_entry(preset, registration, relpath, kind, drive_id, size, mtime_ns, sha1) VALUES (" +
			sqlxdb.Quote(preset) + ", " + sqlxdb.Quote(regID) + ", " + sqlxdb.Quote(e.RelPath) + ", " + sqlxdb.Quote(e.Kind) + ", " +
			sqlxdb.Quote(e.DriveID) + ", " + strconv.FormatInt(e.Size, 10) + ", " + strconv.FormatInt(e.MTimeNS, 10) + ", " + sqlxdb.Quote(e.SHA1) +
			") ON CONFLICT(preset, registration, relpath) DO UPDATE SET kind = excluded.kind, drive_id = excluded.drive_id, " +
			"size = excluded.size, mtime_ns = excluded.mtime_ns, sha1 = excluded.sha1")
	}
	return "DELETE FROM gdrive_sync_entry WHERE " + stateWhere(preset, regID) + " AND relpath NOT IN (" + strings.Join(kept, ", ") + ")" + b.String()
}

func stateWhere(preset, regID string) string {
	return "preset = " + sqlxdb.Quote(preset) + " AND registration = " + sqlxdb.Quote(regID)
}

func Delete(preset, regID string) error {
	path := StateFile(preset, regID)
	if sqlxdb.UsesDB(path) {
		return sqlxdb.Exec("DELETE FROM gdrive_sync_entry WHERE " + stateWhere(preset, regID))
	}
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func sortedKeys(entries map[string]syncer.StateEntry) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
