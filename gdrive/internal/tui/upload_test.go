package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
)

type fakeDrive struct {
	files   []string
	dirs    []string
	trashed []string
}

func (f *fakeDrive) ListChildren(string) ([]driveapi.NavEntry, error) { return nil, nil }

func (f *fakeDrive) FindAvailableName(_, name string) (string, error) { return name, nil }

func (f *fakeDrive) CreateFolder(_, name string) (string, error) {
	f.dirs = append(f.dirs, name)
	return "folder-" + name, nil
}

func (f *fakeDrive) UploadFile(_, name, path string) (string, error) {
	f.files = append(f.files, name+"="+path)
	return "id", nil
}

func (f *fakeDrive) TrashEntry(id string) error {
	f.trashed = append(f.trashed, id)
	return nil
}

func TestUUploadsThePickedFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(file, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	drive := &fakeDrive{}
	m := &model{
		client:   drive,
		parentID: "drive-folder",
		pick:     func() (string, error) { return file, nil },
	}
	if m.handleKey("u") {
		t.Fatal("quit")
	}
	if m.status != "uploaded notes.txt" || len(drive.files) != 1 || drive.files[0] != "notes.txt="+file {
		t.Fatalf("%q %#v", m.status, drive.files)
	}
}

func TestUUploadsADirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "Album")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(sub, "a.txt")
	if err := os.WriteFile(file, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	drive := &fakeDrive{}
	m := &model{client: drive, parentID: "here", pick: func() (string, error) { return sub, nil }}
	if m.handleKey("u") {
		t.Fatal("quit")
	}
	if len(drive.dirs) != 1 || drive.dirs[0] != "Album" {
		t.Fatal(drive.dirs)
	}
	if len(drive.files) != 1 || drive.files[0] != "a.txt="+file {
		t.Fatal(drive.files)
	}
}

func TestUCancelDoesNotUpload(t *testing.T) {
	drive := &fakeDrive{}
	m := &model{client: drive, pick: func() (string, error) { return "", nil }}
	if m.handleKey("u") || m.status != "upload cancelled" || len(drive.files) != 0 {
		t.Fatalf("%q %#v", m.status, drive.files)
	}
}

func TestUReportsAPickerError(t *testing.T) {
	m := &model{pick: func() (string, error) { return "", errors.New("picker failed") }}
	if m.handleKey("u") || m.status != "picker failed" {
		t.Fatal(m.status)
	}
}
