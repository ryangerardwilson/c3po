package syncer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ryangerardwilson/c3po/gdrive/internal/config"
	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
)

type SyncSummary struct {
	Created int
	Updated int
	Moved   int
	Deleted int
}

// syncRun is one local-authoritative pass over a registration.
type syncRun struct {
	drive        driveapi.Client
	localRoot    string
	remoteRootID string
	previous     map[string]StateEntry
	local        map[string]LocalEntry
	remote       map[string]driveapi.RemoteEntry
	dirIDs       map[string]string
	renameMap    map[string]string
	removedFiles map[string]bool
	current      map[string]StateEntry
	summary      SyncSummary
}

// SyncRegistration makes Drive match the local folder, including local
// deletes and content-preserving renames, then saves the sync state.
func SyncRegistration(preset string, registration *config.Registration, drive driveapi.Client, backupRootName string) (SyncSummary, error) {
	localRoot := registration.LocalDir
	info, err := os.Stat(localRoot)
	if err != nil || !info.IsDir() {
		return SyncSummary{}, fmt.Errorf("missing local dir: %s", localRoot)
	}
	previous, err := LoadState(preset, registration.ID)
	if err != nil {
		return SyncSummary{}, err
	}
	localEntries, err := ScanLocalTree(localRoot, previous)
	if err != nil {
		return SyncSummary{}, err
	}
	remoteRootID, err := ensureRemoteRoot(registration, drive, backupRootName)
	if err != nil {
		return SyncSummary{}, err
	}
	remoteEntries, err := drive.ListTree(remoteRootID)
	if err != nil {
		return SyncSummary{}, err
	}
	run := &syncRun{
		drive: drive, localRoot: localRoot, remoteRootID: remoteRootID,
		previous: previous, local: localEntries, remote: remoteEntries,
		current: map[string]StateEntry{},
	}
	if err := run.execute(); err != nil {
		return SyncSummary{}, err
	}
	if err := SaveState(preset, registration.ID, run.current); err != nil {
		return SyncSummary{}, err
	}
	return run.summary, nil
}

func ensureRemoteRoot(registration *config.Registration, drive driveapi.Client, backupRootName string) (string, error) {
	if registration.RemoteRootID != "" {
		return registration.RemoteRootID, nil
	}
	remoteRootID, err := drive.EnsureDrivePath(backupRootName + "/" + registration.DrivePath)
	if err != nil {
		return "", err
	}
	registration.RemoteRootID = remoteRootID
	return remoteRootID, nil
}

func (r *syncRun) execute() error {
	steps := []func() error{
		r.createDirs,
		r.applyRenames,
		r.syncFiles,
		r.deleteRemovedFiles,
		r.deleteRemoteExtras,
		r.recordDirs,
		r.deleteRemovedDirs,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

func (r *syncRun) createDirs() error {
	r.dirIDs = map[string]string{"": r.remoteRootID}
	for relpath, entry := range r.remote {
		if entry.IsDir() {
			r.dirIDs[relpath] = entry.ID
		}
	}
	for _, relpath := range sortPathsDeep(filterLocalKind(r.local, "dir"), false) {
		if existing, ok := r.remote[relpath]; ok && existing.IsDir() {
			r.dirIDs[relpath] = existing.ID
			continue
		}
		driveID, err := r.drive.CreateFolder(r.dirIDs[ParentRelPath(relpath)], filepath.Base(relpath))
		if err != nil {
			return err
		}
		r.dirIDs[relpath] = driveID
		r.summary.Created++
	}
	return nil
}

func (r *syncRun) applyRenames() error {
	r.renameMap = BuildRenameMap(r.previous, r.local)
	r.removedFiles = map[string]bool{}
	for path, entry := range r.previous {
		if entry.Kind != "file" {
			continue
		}
		_, exists := r.local[path]
		_, renamed := r.renameMap[path]
		if !exists && !renamed {
			r.removedFiles[path] = true
		}
	}
	for oldPath, newPath := range r.renameMap {
		oldState := r.previous[oldPath]
		parentID := r.dirIDs[ParentRelPath(newPath)]
		if err := r.drive.MoveEntry(oldState.DriveID, parentID, filepath.Base(newPath), r.previousParentID(oldState)); err != nil {
			return err
		}
		r.summary.Moved++
		r.current[newPath] = fileState(newPath, oldState.DriveID, r.local[newPath])
	}
	return nil
}

func (r *syncRun) previousParentID(oldState StateEntry) string {
	if oldState.ParentRelPath == "" {
		return r.remoteRootID
	}
	if id := r.dirIDs[oldState.ParentRelPath]; id != "" {
		return id
	}
	if parentState, ok := r.previous[oldState.ParentRelPath]; ok {
		return parentState.DriveID
	}
	return r.remoteRootID
}

func (r *syncRun) syncFiles() error {
	for _, relpath := range sortPathsDeep(filterLocalKind(r.local, "file"), false) {
		if _, ok := r.current[relpath]; ok {
			continue
		}
		if err := r.syncFile(relpath); err != nil {
			return err
		}
	}
	return nil
}

func (r *syncRun) syncFile(relpath string) error {
	localEntry := r.local[relpath]
	existing, hasExisting := r.remote[relpath]
	if hasExisting && existing.IsDir() {
		if err := r.drive.DeleteEntry(existing.ID); err != nil {
			return err
		}
		r.summary.Deleted++
		hasExisting = false
	}
	existingFileID := ""
	if hasExisting {
		existingFileID = existing.ID
	}
	var driveID string
	var err error
	if prev, ok := r.previous[relpath]; ok && prev.Kind == "file" {
		driveID, err = r.syncKnownFile(relpath, prev, existingFileID)
	} else {
		driveID, err = r.syncNewFile(relpath, existingFileID)
	}
	if err != nil {
		return err
	}
	r.current[relpath] = fileState(relpath, driveID, localEntry)
	return nil
}

// syncNewFile handles a file the state has not seen: it overwrites a
// remote file at the same path, or uploads a new one.
func (r *syncRun) syncNewFile(relpath, existingFileID string) (string, error) {
	if existingFileID != "" {
		if err := r.drive.UpdateFile(existingFileID, r.localPath(relpath)); err != nil {
			return "", err
		}
		r.summary.Updated++
		return existingFileID, nil
	}
	return r.uploadNew(relpath)
}

func (r *syncRun) uploadNew(relpath string) (string, error) {
	parentID := r.dirIDs[ParentRelPath(relpath)]
	driveID, err := r.drive.UploadFile(parentID, filepath.Base(relpath), r.localPath(relpath))
	if err != nil {
		return "", err
	}
	r.summary.Created++
	return driveID, nil
}

// syncKnownFile handles a file the state recorded: it re-uploads a file
// missing from Drive, updates changed content, and fixes its place.
func (r *syncRun) syncKnownFile(relpath string, prev StateEntry, existingFileID string) (string, error) {
	localEntry := r.local[relpath]
	driveID := existingFileID
	if driveID == "" {
		id, err := r.uploadNew(relpath)
		if err != nil {
			return "", err
		}
		driveID = id
	} else if driveID != prev.DriveID || localEntry.Size != prev.Size || localEntry.SHA1 != prev.SHA1 {
		if err := r.drive.UpdateFile(driveID, r.localPath(relpath)); err != nil {
			return "", err
		}
		r.summary.Updated++
	}
	return driveID, r.relocateKnownFile(relpath, prev, driveID)
}

func (r *syncRun) relocateKnownFile(relpath string, prev StateEntry, driveID string) error {
	name := filepath.Base(relpath)
	if prev.ParentRelPath != ParentRelPath(relpath) {
		oldParentID := r.remoteRootID
		if prev.ParentRelPath != "" {
			oldParentID = r.dirIDs[prev.ParentRelPath]
		}
		if err := r.drive.MoveEntry(driveID, r.dirIDs[ParentRelPath(relpath)], name, oldParentID); err != nil {
			return err
		}
		r.summary.Moved++
		return nil
	}
	if name == filepath.Base(prev.RelPath) {
		return nil
	}
	if err := r.drive.RenameEntry(driveID, name); err != nil {
		return err
	}
	r.summary.Moved++
	return nil
}

func (r *syncRun) deleteRemovedFiles() error {
	for _, relpath := range sortPathsDeep(mapKeys(r.removedFiles), true) {
		if err := r.deleteID(r.previous[relpath].DriveID); err != nil {
			return err
		}
	}
	return nil
}

// deleteRemoteExtras removes remote entries that are neither local nor
// known to the previous state.
func (r *syncRun) deleteRemoteExtras() error {
	extras := []string{}
	for path := range r.remote {
		_, isLocal := r.local[path]
		_, existedBefore := r.previous[path]
		if !isLocal && !existedBefore {
			extras = append(extras, path)
		}
	}
	for _, relpath := range sortPathsDeep(extras, true) {
		if err := r.deleteID(r.remote[relpath].ID); err != nil {
			return err
		}
	}
	return nil
}

func (r *syncRun) recordDirs() error {
	for _, relpath := range sortPathsDeep(filterLocalKind(r.local, "dir"), false) {
		localEntry := r.local[relpath]
		r.current[relpath] = StateEntry{RelPath: relpath, Kind: "dir", DriveID: r.dirIDs[relpath], ParentRelPath: ParentRelPath(relpath), MTimeNS: localEntry.MTimeNS}
	}
	return nil
}

func (r *syncRun) deleteRemovedDirs() error {
	removed := []string{}
	for path, entry := range r.previous {
		if _, exists := r.local[path]; entry.Kind == "dir" && !exists {
			removed = append(removed, path)
		}
	}
	for _, relpath := range sortPathsDeep(removed, true) {
		if err := r.deleteID(r.previous[relpath].DriveID); err != nil {
			return err
		}
	}
	return nil
}

func (r *syncRun) deleteID(driveID string) error {
	if err := r.drive.DeleteEntry(driveID); err != nil {
		return err
	}
	r.summary.Deleted++
	return nil
}

func (r *syncRun) localPath(relpath string) string {
	return filepath.Join(r.localRoot, filepath.FromSlash(relpath))
}

func fileState(relpath, driveID string, local LocalEntry) StateEntry {
	return StateEntry{RelPath: relpath, Kind: "file", DriveID: driveID, ParentRelPath: ParentRelPath(relpath), Size: local.Size, MTimeNS: local.MTimeNS, SHA1: local.SHA1}
}

func filterLocalKind(entries map[string]LocalEntry, kind string) []string {
	paths := []string{}
	for path, entry := range entries {
		if entry.Kind == kind {
			paths = append(paths, path)
		}
	}
	return paths
}
