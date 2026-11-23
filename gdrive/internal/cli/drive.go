package cli

import (
	"context"
	"time"

	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
	"github.com/ryangerardwilson/c3po/gdrive/internal/syncer"
)

// screenRequestWait bounds one Drive request from the screen. An upload
// sends at most one 8 MiB chunk per request.
const screenRequestWait = 2 * time.Minute

// screenDrive is the Drive surface the screen uses. Each call ends when its
// ctx does.
type screenDrive struct {
	client *driveapi.DriveClient
}

func (d screenDrive) ListChildren(ctx context.Context, parentID string) ([]driveapi.NavEntry, error) {
	return d.client.WithContext(ctx).ListChildren(parentID)
}

func (d screenDrive) FindAvailableName(ctx context.Context, parentID, name string) (string, error) {
	return d.client.WithContext(ctx).FindAvailableName(parentID, name)
}

func (d screenDrive) CreateFolder(ctx context.Context, parentID, name string) (string, error) {
	return d.client.WithContext(ctx).CreateFolder(parentID, name)
}

func (d screenDrive) UploadFile(ctx context.Context, parentID, name, filePath string) (string, error) {
	return d.client.WithContext(ctx).UploadFile(parentID, name, filePath)
}

func (d screenDrive) TrashEntry(ctx context.Context, fileID string) error {
	return d.client.WithContext(ctx).TrashEntry(fileID)
}

// syncDrive presents a Drive client to the syncer in syncer types. The
// other syncer.SyncRemote and syncer.RestoreRemote methods come from the
// embedded client unchanged.
type syncDrive struct {
	*driveapi.DriveClient
}

func (d syncDrive) ListTree(rootID string) (map[string]syncer.RemoteEntry, error) {
	entries, err := d.DriveClient.ListTree(rootID)
	if err != nil {
		return nil, err
	}
	return syncEntries(entries), nil
}

func (d syncDrive) Download(entry syncer.RemoteEntry, target string) (string, error) {
	return d.DriveClient.DownloadEntry(navEntry(entry), target)
}

func syncEntries(entries map[string]driveapi.RemoteEntry) map[string]syncer.RemoteEntry {
	out := make(map[string]syncer.RemoteEntry, len(entries))
	for rel, entry := range entries {
		out[rel] = syncer.RemoteEntry{
			ID: entry.ID, RelPath: entry.RelPath, Name: entry.Name, ParentID: entry.ParentID,
			ContentType: entry.MimeType, IsDir: entry.IsDir(),
		}
	}
	return out
}

func navEntry(entry syncer.RemoteEntry) driveapi.NavEntry {
	return driveapi.NavEntry{ID: entry.ID, Name: entry.Name, MimeType: entry.ContentType, ParentID: entry.ParentID}
}
