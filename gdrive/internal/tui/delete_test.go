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
	if press(t, m, "d") || press(t, m, "d") {
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
	if press(t, m, "d") {
		t.Fatal("quit")
	}
	if press(t, m, "j") || m.cursor != 1 || len(drive.trashed) != 0 || m.armed {
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
	if press(t, m, "d") || press(t, m, "d") || len(drive.trashed) != 0 || m.status != "dd deletes a file" {
		t.Fatalf("%q %#v", m.status, drive.trashed)
	}
}

func TestEscCancelsDelete(t *testing.T) {
	drive := &fakeDrive{}
	m := &model{
		client:  drive,
		entries: []driveapi.NavEntry{{ID: "a", Name: "a.txt", MimeType: "text/plain"}},
	}
	press(t, m, "d")
	if press(t, m, "esc") || m.armed || m.status != "" || len(drive.trashed) != 0 {
		t.Fatalf("armed %v %q %#v", m.armed, m.status, drive.trashed)
	}
}

func TestJKWrapsTheFileList(t *testing.T) {
	m := &model{entries: []driveapi.NavEntry{
		{ID: "a", Name: "a.txt"},
		{ID: "b", Name: "b.txt"},
	}}
	if press(t, m, "k") || m.cursor != 1 {
		t.Fatalf("k %d", m.cursor)
	}
	if press(t, m, "j") || m.cursor != 0 {
		t.Fatalf("j %d", m.cursor)
	}
	if press(t, m, "G") || m.cursor != 1 || press(t, m, "g") || m.cursor != 0 {
		t.Fatalf("ends %d", m.cursor)
	}
	empty := &model{}
	empty.handleKey("j")
	empty.handleKey("k")
	if empty.cursor != 0 {
		t.Fatalf("empty %d", empty.cursor)
	}
}
