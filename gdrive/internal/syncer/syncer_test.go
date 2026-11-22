package syncer

import (
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// fakeDrive is the remote side in memory. Download writes into local.
type fakeDrive struct {
	tree     map[string]RemoteEntry
	local    *fakeLocal
	deleted  []string
	uploaded []string
	updated  []string
}

func (f *fakeDrive) EnsureDrivePath(path string) (string, error) { return "root-id", nil }
func (f *fakeDrive) ListTree(rootID string) (map[string]RemoteEntry, error) {
	return f.tree, nil
}
func (f *fakeDrive) CreateFolder(parentID, name string) (string, error) {
	f.uploaded = append(f.uploaded, "dir:"+name)
	return "created-" + name, nil
}
func (f *fakeDrive) UploadFile(parentID, name, source string) (string, error) {
	f.uploaded = append(f.uploaded, "file:"+name+"="+source)
	return "uploaded-" + name, nil
}
func (f *fakeDrive) UpdateFile(fileID, source string) error {
	f.updated = append(f.updated, fileID)
	return nil
}
func (f *fakeDrive) MoveEntry(fileID, newParentID, newName, oldParentID string) error {
	f.updated = append(f.updated, "move:"+fileID+":"+newName)
	return nil
}
func (f *fakeDrive) RenameEntry(fileID, newName string) error {
	f.updated = append(f.updated, "rename:"+fileID+":"+newName)
	return nil
}
func (f *fakeDrive) DeleteEntry(fileID string) error {
	f.deleted = append(f.deleted, fileID)
	return nil
}
func (f *fakeDrive) Download(entry RemoteEntry, target string) (string, error) {
	rel, _ := f.local.RelPath(target)
	f.local.entries[rel] = LocalEntry{RelPath: rel, Kind: "file", Size: int64(len("downloaded " + entry.Name)), MTimeNS: 7}
	return target, nil
}

func restoreTree() map[string]RemoteEntry {
	return map[string]RemoteEntry{
		"Album":          {ID: "dir-1", RelPath: "Album", Name: "Album", ParentID: "root-id", IsDir: true},
		"Album/song.mp3": {ID: "file-1", RelPath: "Album/song.mp3", Name: "song.mp3", ParentID: "dir-1", ContentType: "audio/mpeg"},
		"book.pdf":       {ID: "file-2", RelPath: "book.pdf", Name: "book.pdf", ParentID: "root-id", ContentType: "application/pdf"},
	}
}

// fakeLocal is the registered folder in memory under the locator /local.
// The empty relpath is the folder itself and exists once made.
type fakeLocal struct {
	entries map[string]LocalEntry
	hashed  []string
}

func newFakeLocal(entries ...LocalEntry) *fakeLocal {
	f := &fakeLocal{entries: map[string]LocalEntry{"": {Kind: "dir"}}}
	for _, entry := range entries {
		f.entries[entry.RelPath] = entry
	}
	return f
}

func (f *fakeLocal) Stat(relpath string) (LocalEntry, error) {
	entry, ok := f.entries[relpath]
	if !ok {
		return LocalEntry{}, &fs.PathError{Op: "stat", Path: f.Locate(relpath), Err: fs.ErrNotExist}
	}
	return entry, nil
}
func (f *fakeLocal) Hash(relpath string) (string, error) {
	f.hashed = append(f.hashed, relpath)
	return "sha-" + relpath, nil
}
func (f *fakeLocal) Locate(relpath string) string {
	if relpath == "" {
		return "/local"
	}
	return "/local/" + relpath
}
func (f *fakeLocal) Scan() (map[string]LocalEntry, error) {
	out := map[string]LocalEntry{}
	for rel, entry := range f.entries {
		if rel != "" {
			out[rel] = entry
		}
	}
	return out, nil
}
func (f *fakeLocal) MakeDir(relpath string) error {
	for rel := relpath; ; rel = ParentRelPath(rel) {
		if _, ok := f.entries[rel]; !ok {
			f.entries[rel] = LocalEntry{RelPath: rel, Kind: "dir", MTimeNS: 3}
		}
		if rel == "" {
			return nil
		}
	}
}
func (f *fakeLocal) RelPath(locator string) (string, bool) {
	rel, ok := strings.CutPrefix(locator, "/local/")
	return rel, ok
}

func TestParentRelPath(t *testing.T) {
	if ParentRelPath("a/b/c.txt") != "a/b" {
		t.Fatal("unexpected parent")
	}
	if ParentRelPath("file.txt") != "" {
		t.Fatal("unexpected root parent")
	}
}

func TestBuildRenameMap(t *testing.T) {
	old := map[string]StateEntry{"old.txt": {RelPath: "old.txt", Kind: "file", DriveID: "id-1", Size: 5, SHA1: "abc"}}
	newEntries := map[string]LocalEntry{"new.txt": {RelPath: "new.txt", Kind: "file", Size: 5, SHA1: "abc"}}
	got := BuildRenameMap(old, newEntries)
	if got["old.txt"] != "new.txt" {
		t.Fatalf("rename map = %#v", got)
	}
}

func TestRestoreSeedsStateWithoutDeleting(t *testing.T) {
	reg := Registration{ID: "1", DrivePath: "Music"}
	local := &fakeLocal{entries: map[string]LocalEntry{}}
	drive := &fakeDrive{tree: restoreTree(), local: local}
	store := memStore{}
	summary, err := RestoreRegistrationFromRemote("1", &reg, local, drive, store, "Backups")
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.Load("1", "1")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Downloaded != 2 || summary.DirsCreated != 1 || len(drive.deleted) != 0 {
		t.Fatalf("summary=%#v deleted=%#v", summary, drive.deleted)
	}
	if _, ok := state["Album/song.mp3"]; !ok {
		t.Fatalf("state missing song: %#v", state)
	}
	if reg.RemoteRootID != "root-id" || state["Album/song.mp3"].SHA1 != "sha-Album/song.mp3" || state["Album"].DriveID != "dir-1" {
		t.Fatalf("reg=%#v state=%#v", reg, state)
	}
}

func TestRestoreSkipsExistingAndRefusesFileOverDir(t *testing.T) {
	reg := Registration{ID: "1", RemoteRootID: "root-id"}
	local := newFakeLocal(LocalEntry{RelPath: "book.pdf", Kind: "file", Size: 1})
	drive := &fakeDrive{tree: restoreTree(), local: local}
	summary, err := RestoreRegistrationFromRemote("1", &reg, local, drive, memStore{}, "Backups")
	if err != nil || summary.SkippedExisting != 1 || summary.Downloaded != 1 || summary.StateEntries != 2 {
		t.Fatalf("summary=%#v err=%v", summary, err)
	}
	local = newFakeLocal(LocalEntry{RelPath: "book.pdf", Kind: "dir"})
	drive.local = local
	_, err = RestoreRegistrationFromRemote("1", &reg, local, drive, memStore{}, "Backups")
	if err == nil || err.Error() != "cannot restore file over directory: /local/book.pdf" {
		t.Fatalf("err=%v", err)
	}
}

func TestSyncMissingLocalDir(t *testing.T) {
	local := &fakeLocal{entries: map[string]LocalEntry{}}
	_, err := SyncRegistration("1", &Registration{ID: "1"}, local, &fakeDrive{}, memStore{}, "Backups")
	if err == nil || err.Error() != "missing local dir: /local" {
		t.Fatalf("err=%v", err)
	}
}

func TestSyncUploadsRenamesAndDeletes(t *testing.T) {
	reg := Registration{ID: "1", DrivePath: "Docs"}
	local := newFakeLocal(
		LocalEntry{RelPath: "notes", Kind: "dir", MTimeNS: 1},
		LocalEntry{RelPath: "notes/new.txt", Kind: "file", Size: 4, MTimeNS: 2},
		LocalEntry{RelPath: "kept.txt", Kind: "file", Size: 3, MTimeNS: 5},
		LocalEntry{RelPath: "renamed.txt", Kind: "file", Size: 9, MTimeNS: 6},
	)
	store := memStore{"1-1": {
		"kept.txt": {RelPath: "kept.txt", Kind: "file", DriveID: "f-kept", Size: 3, MTimeNS: 5, SHA1: "old-hash"},
		"old.txt":  {RelPath: "old.txt", Kind: "file", DriveID: "f-old", Size: 9, SHA1: "sha-renamed.txt"},
		"gone.txt": {RelPath: "gone.txt", Kind: "file", DriveID: "f-gone", Size: 1, SHA1: "x"},
	}}
	drive := &fakeDrive{tree: map[string]RemoteEntry{
		"kept.txt":  {ID: "f-kept", RelPath: "kept.txt", Name: "kept.txt", ParentID: "root-id"},
		"old.txt":   {ID: "f-old", RelPath: "old.txt", Name: "old.txt", ParentID: "root-id"},
		"gone.txt":  {ID: "f-gone", RelPath: "gone.txt", Name: "gone.txt", ParentID: "root-id"},
		"extra.txt": {ID: "f-extra", RelPath: "extra.txt", Name: "extra.txt", ParentID: "root-id"},
	}}
	summary, err := SyncRegistration("1", &reg, local, drive, store, "Backups")
	if err != nil {
		t.Fatal(err)
	}
	want := SyncSummary{Created: 2, Moved: 1, Deleted: 2}
	if summary != want || reg.RemoteRootID != "root-id" {
		t.Fatalf("summary=%#v reg=%#v", summary, reg)
	}
	sort.Strings(local.hashed)
	if !reflect.DeepEqual(local.hashed, []string{"notes/new.txt", "renamed.txt"}) {
		t.Fatalf("hashed %#v (kept.txt must reuse its recorded hash)", local.hashed)
	}
	if !reflect.DeepEqual(drive.uploaded, []string{"dir:notes", "file:new.txt=/local/notes/new.txt"}) {
		t.Fatalf("uploaded %#v", drive.uploaded)
	}
	if !reflect.DeepEqual(drive.updated, []string{"move:f-old:renamed.txt"}) {
		t.Fatalf("updated %#v", drive.updated)
	}
	sort.Strings(drive.deleted)
	if !reflect.DeepEqual(drive.deleted, []string{"f-extra", "f-gone"}) {
		t.Fatalf("deleted %#v", drive.deleted)
	}
	state, _ := store.Load("1", "1")
	if len(state) != 4 || state["renamed.txt"].DriveID != "f-old" || state["kept.txt"].SHA1 != "old-hash" || state["notes"].DriveID != "created-notes" {
		t.Fatalf("state %#v", state)
	}
}

// memStore is a StateStore in memory, keyed by preset and registration.
type memStore map[string]map[string]StateEntry

func (m memStore) Load(preset, regID string) (map[string]StateEntry, error) {
	out := map[string]StateEntry{}
	for key, entry := range m[preset+"-"+regID] {
		out[key] = entry
	}
	return out, nil
}

func (m memStore) Save(preset, regID string, entries map[string]StateEntry) error {
	m[preset+"-"+regID] = entries
	return nil
}
