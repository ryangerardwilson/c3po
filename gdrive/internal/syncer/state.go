// Package syncer makes Drive match a registered local folder, and seeds
// that folder from Drive on restore.
package syncer

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/gdrive/internal/paths"
	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

type LocalEntry struct {
	RelPath string `json:"relpath"`
	Kind    string `json:"kind"`
	Size    int64  `json:"size"`
	MTimeNS int64  `json:"mtime_ns"`
	SHA1    string `json:"sha1,omitempty"`
}

type StateEntry struct {
	RelPath       string `json:"relpath"`
	Kind          string `json:"kind"`
	DriveID       string `json:"drive_id"`
	ParentRelPath string `json:"parent_relpath"`
	Size          int64  `json:"size"`
	MTimeNS       int64  `json:"mtime_ns"`
	SHA1          string `json:"sha1,omitempty"`
}

type statePayload struct {
	Entries []StateEntry `json:"entries"`
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

func LoadState(preset, regID string) (map[string]StateEntry, error) {
	path := StateFile(preset, regID)
	data, err := sqlxdb.Read("gdrive", "state", sqlxdb.Key(path), path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]StateEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	payload := statePayload{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	result := map[string]StateEntry{}
	for _, entry := range payload.Entries {
		result[entry.RelPath] = entry
	}
	return result, nil
}

func SaveState(preset, regID string, entries map[string]StateEntry) error {
	path := StateFile(preset, regID)
	keys := sortedMapKeys(entries)
	payload := statePayload{Entries: make([]StateEntry, 0, len(keys))}
	for _, key := range keys {
		payload.Entries = append(payload.Entries, entries[key])
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return sqlxdb.Write("gdrive", "state", sqlxdb.Key(path), path, data, 0o600)
}

func DeleteState(preset, regID string) error {
	path := StateFile(preset, regID)
	return sqlxdb.Remove("gdrive", "state", sqlxdb.Key(path), path)
}

func SHA1File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha1.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func ParentRelPath(relpath string) string {
	if !strings.Contains(relpath, "/") {
		return ""
	}
	return relpath[:strings.LastIndex(relpath, "/")]
}

func ScanLocalTree(root string, previous map[string]StateEntry) (map[string]LocalEntry, error) {
	result := map[string]LocalEntry{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		return scanEntry(root, path, d, previous, result)
	})
	return result, err
}

func scanEntry(root, path string, d os.DirEntry, previous map[string]StateEntry, result map[string]LocalEntry) error {
	info, err := d.Info()
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if d.IsDir() {
		result[rel] = LocalEntry{RelPath: rel, Kind: "dir", MTimeNS: info.ModTime().UnixNano()}
		return nil
	}
	hash, err := fileHash(path, info, previous[rel])
	if err != nil {
		return err
	}
	result[rel] = LocalEntry{RelPath: rel, Kind: "file", Size: info.Size(), MTimeNS: info.ModTime().UnixNano(), SHA1: hash}
	return nil
}

// fileHash reuses the recorded hash when size and mtime are unchanged.
func fileHash(path string, info os.FileInfo, old StateEntry) (string, error) {
	if old.Kind == "file" && old.Size == info.Size() && old.MTimeNS == info.ModTime().UnixNano() && old.SHA1 != "" {
		return old.SHA1, nil
	}
	return SHA1File(path)
}

type fileSig struct {
	Size int64
	SHA1 string
}

func BuildRenameMap(oldEntries map[string]StateEntry, newEntries map[string]LocalEntry) map[string]string {
	bySig := map[fileSig][]LocalEntry{}
	for path, entry := range newEntries {
		if _, ok := oldEntries[path]; ok || entry.Kind != "file" {
			continue
		}
		key := fileSig{Size: entry.Size, SHA1: entry.SHA1}
		bySig[key] = append(bySig[key], entry)
	}
	renameMap := map[string]string{}
	for path, entry := range oldEntries {
		if _, ok := newEntries[path]; ok || entry.Kind != "file" {
			continue
		}
		key := fileSig{Size: entry.Size, SHA1: entry.SHA1}
		bucket := bySig[key]
		if len(bucket) == 0 {
			continue
		}
		renameMap[path] = bucket[0].RelPath
		bySig[key] = bucket[1:]
	}
	return renameMap
}

func sortPathsDeep(paths []string, reverse bool) []string {
	out := append([]string(nil), paths...)
	sort.SliceStable(out, func(i, j int) bool {
		leftDepth := strings.Count(out[i], "/")
		rightDepth := strings.Count(out[j], "/")
		if leftDepth != rightDepth {
			if reverse {
				return leftDepth > rightDepth
			}
			return leftDepth < rightDepth
		}
		if reverse {
			return out[i] > out[j]
		}
		return out[i] < out[j]
	})
	return out
}

func sortedMapKeys(entries map[string]StateEntry) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func mapKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
