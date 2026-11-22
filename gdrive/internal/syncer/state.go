// Package syncer makes Drive match a registered local folder, and seeds
// that folder from Drive on restore.
package syncer

import (
	"sort"
	"strings"
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
	ParentRelPath string `json:"-"` // derived from RelPath on load; not stored
	Size          int64  `json:"size"`
	MTimeNS       int64  `json:"mtime_ns"`
	SHA1          string `json:"sha1,omitempty"`
}

// StateStore keeps one registration's sync entries. Saving replaces the
// stored set with entries.
type StateStore interface {
	Load(preset, regID string) (map[string]StateEntry, error)
	Save(preset, regID string, entries map[string]StateEntry) error
}

func ParentRelPath(relpath string) string {
	if !strings.Contains(relpath, "/") {
		return ""
	}
	return relpath[:strings.LastIndex(relpath, "/")]
}

// scanLocal lists the local folder and hashes each file, reusing the
// recorded hash when size and mtime are unchanged.
func scanLocal(local LocalSource, previous map[string]StateEntry) (map[string]LocalEntry, error) {
	entries, err := local.Scan()
	if err != nil {
		return nil, err
	}
	for _, relpath := range sortPathsDeep(filterLocalKind(entries, "file"), false) {
		entry := entries[relpath]
		hash, err := fileHash(local, entry, previous[relpath])
		if err != nil {
			return nil, err
		}
		entry.SHA1 = hash
		entries[relpath] = entry
	}
	return entries, nil
}

func fileHash(local LocalFiles, entry LocalEntry, old StateEntry) (string, error) {
	if old.Kind == "file" && old.Size == entry.Size && old.MTimeNS == entry.MTimeNS && old.SHA1 != "" {
		return old.SHA1, nil
	}
	return local.Hash(entry.RelPath)
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

func mapKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
