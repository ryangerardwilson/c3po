package syncer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/gdrive/internal/config"
	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
)

type RestoreSummary struct {
	Downloaded      int
	DirsCreated     int
	SkippedExisting int
	StateEntries    int
}

type restoreRun struct {
	drive     driveapi.Client
	localRoot string
	remote    map[string]driveapi.RemoteEntry
	current   map[string]StateEntry
	summary   RestoreSummary
}

// RestoreRegistrationFromRemote downloads remote content that is missing
// locally, never deletes, and seeds the sync state from the result.
func RestoreRegistrationFromRemote(preset string, registration *config.Registration, drive driveapi.Client, backupRootName string) (RestoreSummary, error) {
	localRoot := registration.LocalDir
	if err := os.MkdirAll(localRoot, 0o755); err != nil {
		return RestoreSummary{}, err
	}
	remoteRootID, err := ensureRemoteRoot(registration, drive, backupRootName)
	if err != nil {
		return RestoreSummary{}, err
	}
	remoteEntries, err := drive.ListTree(remoteRootID)
	if err != nil {
		return RestoreSummary{}, err
	}
	run := &restoreRun{drive: drive, localRoot: localRoot, remote: remoteEntries, current: map[string]StateEntry{}}
	if err := run.restoreDirs(); err != nil {
		return RestoreSummary{}, err
	}
	if err := run.restoreFiles(); err != nil {
		return RestoreSummary{}, err
	}
	run.summary.StateEntries = len(run.current)
	if err := SaveState(preset, registration.ID, run.current); err != nil {
		return RestoreSummary{}, err
	}
	return run.summary, nil
}

func (r *restoreRun) restoreDirs() error {
	for _, relpath := range sortPathsDeep(filterRemoteKind(r.remote, true), false) {
		target := filepath.Join(r.localRoot, filepath.FromSlash(relpath))
		if _, err := os.Stat(target); os.IsNotExist(err) {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			r.summary.DirsCreated++
		}
		info, err := os.Stat(target)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("cannot restore directory over file: %s", target)
		}
		r.current[relpath] = StateEntry{RelPath: relpath, Kind: "dir", DriveID: r.remote[relpath].ID, ParentRelPath: ParentRelPath(relpath), MTimeNS: info.ModTime().UnixNano()}
	}
	return nil
}

func (r *restoreRun) restoreFiles() error {
	for _, relpath := range sortPathsDeep(filterRemoteKind(r.remote, false), false) {
		if err := r.restoreFile(relpath); err != nil {
			return err
		}
	}
	return nil
}

func (r *restoreRun) restoreFile(relpath string) error {
	entry := r.remote[relpath]
	target := filepath.Join(r.localRoot, filepath.FromSlash(relpath))
	if info, err := os.Stat(target); err == nil {
		if info.IsDir() {
			return fmt.Errorf("cannot restore file over directory: %s", target)
		}
		r.summary.SkippedExisting++
		return nil
	}
	downloaded, err := r.drive.DownloadEntry(driveapi.NavEntry{ID: entry.ID, Name: entry.Name, MimeType: entry.MimeType, ParentID: entry.ParentID}, target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(r.localRoot, downloaded)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("download escaped local root: %s", downloaded)
	}
	rel = filepath.ToSlash(rel)
	info, err := os.Stat(downloaded)
	if err != nil {
		return err
	}
	hash, err := SHA1File(downloaded)
	if err != nil {
		return err
	}
	r.current[rel] = StateEntry{RelPath: rel, Kind: "file", DriveID: entry.ID, ParentRelPath: ParentRelPath(rel), Size: info.Size(), MTimeNS: info.ModTime().UnixNano(), SHA1: hash}
	r.summary.Downloaded++
	return nil
}

func filterRemoteKind(entries map[string]driveapi.RemoteEntry, dirs bool) []string {
	paths := []string{}
	for path, entry := range entries {
		if entry.IsDir() == dirs {
			paths = append(paths, path)
		}
	}
	return paths
}
