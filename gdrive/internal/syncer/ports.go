package syncer

// Registration is the part of a registered folder the syncer reads. Sync
// and restore fill RemoteRootID when it is empty.
type Registration struct {
	ID           string
	DrivePath    string
	RemoteRootID string
}

// RemoteEntry is one file or folder under the remote root, keyed by its
// slash-separated path from that root.
type RemoteEntry struct {
	ID          string
	RelPath     string
	Name        string
	ParentID    string
	ContentType string
	IsDir       bool
}

// RemoteTree finds the remote root and lists everything under it.
type RemoteTree interface {
	EnsureDrivePath(drivePath string) (string, error)
	ListTree(rootID string) (map[string]RemoteEntry, error)
}

// RemoteUploader puts local folders and file content on the remote side.
// source is the locator LocalFiles.Locate returned.
type RemoteUploader interface {
	CreateFolder(parentID, name string) (string, error)
	UploadFile(parentID, name, source string) (string, error)
	UpdateFile(fileID, source string) error
}

// RemoteOrganizer moves, renames, and deletes remote entries.
type RemoteOrganizer interface {
	MoveEntry(fileID, newParentID, newName, oldParentID string) error
	RenameEntry(fileID, newName string) error
	DeleteEntry(fileID string) error
}

// RemoteDownloader writes one remote file to target and returns the
// locator it wrote, which may carry a different extension.
type RemoteDownloader interface {
	Download(entry RemoteEntry, target string) (string, error)
}

// SyncRemote is the remote side of a sync.
type SyncRemote interface {
	RemoteTree
	RemoteUploader
	RemoteOrganizer
}

// RestoreRemote is the remote side of a restore.
type RestoreRemote interface {
	RemoteTree
	RemoteDownloader
}

// LocalFiles reads entries of the registered folder by relpath. The empty
// relpath is the folder itself. Stat reports a missing entry with an
// error that matches fs.ErrNotExist and leaves SHA1 empty.
type LocalFiles interface {
	Stat(relpath string) (LocalEntry, error)
	Hash(relpath string) (string, error)
	Locate(relpath string) string
}

// LocalSource is the local side of a sync. Scan lists every entry under
// the folder, skipping links, without hashing files.
type LocalSource interface {
	LocalFiles
	Scan() (map[string]LocalEntry, error)
}

// LocalTarget is the local side of a restore. MakeDir creates a folder
// and its parents. RelPath maps a locator back to a relpath and reports
// false when it lies outside the folder.
type LocalTarget interface {
	LocalFiles
	MakeDir(relpath string) error
	RelPath(locator string) (string, bool)
}
