package cli

import (
	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
	"github.com/ryangerardwilson/c3po/gdrive/internal/syncer"
)

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
