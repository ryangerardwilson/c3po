package cli

import (
	"testing"

	"github.com/ryangerardwilson/c3po/gdrive/internal/config"
	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
	"github.com/ryangerardwilson/c3po/gdrive/internal/syncer"
)

func TestSyncEntriesCarryDriveFields(t *testing.T) {
	got := syncEntries(map[string]driveapi.RemoteEntry{
		"Album":          {ID: "dir-1", RelPath: "Album", Name: "Album", ParentID: "root", MimeType: driveapi.FolderMime},
		"Album/doc.docx": {ID: "f-1", RelPath: "Album/doc.docx", Name: "doc", ParentID: "dir-1", MimeType: "application/vnd.google-apps.document"},
	})
	if !got["Album"].IsDir || got["Album"].ID != "dir-1" || got["Album"].ParentID != "root" {
		t.Fatalf("dir = %#v", got["Album"])
	}
	doc := got["Album/doc.docx"]
	if doc.IsDir || doc.RelPath != "Album/doc.docx" || doc.Name != "doc" || doc.ContentType != "application/vnd.google-apps.document" {
		t.Fatalf("doc = %#v", doc)
	}
	nav := navEntry(doc)
	if nav != (driveapi.NavEntry{ID: "f-1", Name: "doc", MimeType: "application/vnd.google-apps.document", ParentID: "dir-1"}) {
		t.Fatalf("nav = %#v", nav)
	}
}

func TestSyncDriveIsBothSyncerRemotes(t *testing.T) {
	var _ syncer.SyncRemote = syncDrive{}
	var _ syncer.RestoreRemote = syncDrive{}
}

func TestSyncTargetKeepsRegistrationFields(t *testing.T) {
	got := syncTarget(&config.Registration{ID: "3", LocalDir: "/m", DrivePath: "Music", RemoteRootID: "r"})
	if got != (syncer.Registration{ID: "3", DrivePath: "Music", RemoteRootID: "r"}) {
		t.Fatalf("%#v", got)
	}
}
