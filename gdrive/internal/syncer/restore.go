package syncer

import (
	"errors"
	"io/fs"
)

type RestoreSummary struct {
	Downloaded      int
	DirsCreated     int
	SkippedExisting int
	StateEntries    int
}

type restoreRun struct {
	drive   RestoreRemote
	local   LocalTarget
	remote  map[string]RemoteEntry
	current map[string]StateEntry
	summary RestoreSummary
}

// RestoreRegistrationFromRemote downloads remote content that is missing
// locally, never deletes, and seeds the sync state from the result.
func RestoreRegistrationFromRemote(preset string, registration *Registration, local LocalTarget, drive RestoreRemote, state StateStore, backupRootName string) (RestoreSummary, error) {
	if err := local.MakeDir(""); err != nil {
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
	run := &restoreRun{drive: drive, local: local, remote: remoteEntries, current: map[string]StateEntry{}}
	if err := run.restoreDirs(); err != nil {
		return RestoreSummary{}, err
	}
	if err := run.restoreFiles(); err != nil {
		return RestoreSummary{}, err
	}
	run.summary.StateEntries = len(run.current)
	if err := state.Save(preset, registration.ID, run.current); err != nil {
		return RestoreSummary{}, err
	}
	return run.summary, nil
}

func (r *restoreRun) restoreDirs() error {
	for _, relpath := range sortPathsDeep(filterRemoteKind(r.remote, true), false) {
		if _, err := r.local.Stat(relpath); errors.Is(err, fs.ErrNotExist) {
			if err := r.local.MakeDir(relpath); err != nil {
				return err
			}
			r.summary.DirsCreated++
		}
		info, err := r.local.Stat(relpath)
		if err != nil {
			return err
		}
		if info.Kind != "dir" {
			return errors.New("cannot restore directory over file: " + r.local.Locate(relpath))
		}
		r.current[relpath] = StateEntry{RelPath: relpath, Kind: "dir", DriveID: r.remote[relpath].ID, ParentRelPath: ParentRelPath(relpath), MTimeNS: info.MTimeNS}
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
	target := r.local.Locate(relpath)
	if info, err := r.local.Stat(relpath); err == nil {
		if info.Kind == "dir" {
			return errors.New("cannot restore file over directory: " + target)
		}
		r.summary.SkippedExisting++
		return nil
	}
	downloaded, err := r.drive.Download(entry, target)
	if err != nil {
		return err
	}
	rel, ok := r.local.RelPath(downloaded)
	if !ok {
		return errors.New("download escaped local root: " + downloaded)
	}
	info, err := r.local.Stat(rel)
	if err != nil {
		return err
	}
	hash, err := r.local.Hash(rel)
	if err != nil {
		return err
	}
	r.current[rel] = StateEntry{RelPath: rel, Kind: "file", DriveID: entry.ID, ParentRelPath: ParentRelPath(rel), Size: info.Size, MTimeNS: info.MTimeNS, SHA1: hash}
	r.summary.Downloaded++
	return nil
}

func filterRemoteKind(entries map[string]RemoteEntry, dirs bool) []string {
	paths := []string{}
	for rel, entry := range entries {
		if entry.IsDir == dirs {
			paths = append(paths, rel)
		}
	}
	return paths
}
