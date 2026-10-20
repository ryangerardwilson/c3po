package tui

import (
	"errors"
	"testing"

	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
)

func TestEnterOpensTheWebLink(t *testing.T) {
	var got string
	m := &model{
		entries: []driveapi.NavEntry{{
			ID: "abc", Name: "notes", MimeType: driveapi.FolderMime,
			WebViewLink: "https://drive.google.com/drive/folders/abc",
		}},
		open:     func(link string) error { got = link; return nil },
		parentID: "root",
	}
	if m.handleKey("enter") {
		t.Fatal("quit")
	}
	if got != "https://drive.google.com/drive/folders/abc" || m.status != "opened notes" || m.parentID != "root" {
		t.Fatalf("%q %q %s", got, m.status, m.parentID)
	}
}

func TestEnterBuildsAFileLink(t *testing.T) {
	var got string
	m := &model{
		entries: []driveapi.NavEntry{{ID: "abc", Name: "a.txt", MimeType: "text/plain"}},
		open:    func(link string) error { got = link; return nil },
	}
	if m.handleKey("enter") {
		t.Fatal("quit")
	}
	if got != "https://drive.google.com/file/d/abc/view" {
		t.Fatal(got)
	}
}

func TestEnterReportsABrowserError(t *testing.T) {
	m := &model{
		entries: []driveapi.NavEntry{{ID: "abc", Name: "a.txt"}},
		open:    func(string) error { return errors.New("xdg-open failed") },
	}
	if m.handleKey("enter") {
		t.Fatal("quit")
	}
	if m.status != "xdg-open failed" || m.err == nil {
		t.Fatal(m.status)
	}
}

func TestFolderURLWithoutAWebLink(t *testing.T) {
	got := itemURL(driveapi.NavEntry{ID: "dir", MimeType: driveapi.FolderMime})
	if got != "https://drive.google.com/drive/folders/dir" {
		t.Fatal(got)
	}
}
