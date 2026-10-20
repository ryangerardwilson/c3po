package tui

import (
	"testing"

	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
)

func TestDDDeletesTheSelectedFile(t *testing.T) {
	drive := &fakeDrive{}
	m := &model{
		client:   drive,
		parentID: "folder",
		entries: []driveapi.NavEntry{
			{ID: "a", Name: "a.txt", MimeType: "text/plain"},
			{ID: "b", Name: "b.txt", MimeType: "text/plain"},
		},
		cursor: 1,
	}
	if m.handleKey("d") || m.handleKey("d") {
		t.Fatal("quit")
	}
	if len(drive.trashed) != 1 || drive.trashed[0] != "b" || m.status != "deleted b.txt" {
		t.Fatalf("%q %#v", m.status, drive.trashed)
	}
}

func TestDThenMotionDoesNotDelete(t *testing.T) {
	drive := &fakeDrive{}
	m := &model{
		client: drive,
		entries: []driveapi.NavEntry{
			{ID: "a", Name: "a.txt", MimeType: "text/plain"},
			{ID: "b", Name: "b.txt", MimeType: "text/plain"},
		},
	}
	if m.handleKey("d") {
		t.Fatal("quit")
	}
	if m.handleKey("j") || m.cursor != 1 || len(drive.trashed) != 0 || m.armed {
		t.Fatalf("cursor %d armed %v %#v", m.cursor, m.armed, drive.trashed)
	}
}

func TestDDLeavesAFolder(t *testing.T) {
	drive := &fakeDrive{}
	m := &model{
		client: drive,
		entries: []driveapi.NavEntry{{
			ID: "dir", Name: "Album", MimeType: driveapi.FolderMime,
		}},
	}
	if m.handleKey("d") || m.handleKey("d") || len(drive.trashed) != 0 || m.status != "dd deletes a file" {
		t.Fatalf("%q %#v", m.status, drive.trashed)
	}
}

func TestEscCancelsDelete(t *testing.T) {
	drive := &fakeDrive{}
	m := &model{
		client:  drive,
		entries: []driveapi.NavEntry{{ID: "a", Name: "a.txt", MimeType: "text/plain"}},
	}
	m.handleKey("d")
	if m.handleKey("esc") || m.armed || m.status != "" || len(drive.trashed) != 0 {
		t.Fatalf("armed %v %q %#v", m.armed, m.status, drive.trashed)
	}
}
