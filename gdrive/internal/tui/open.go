package tui

import (
	"net/url"
	"os/exec"
	"runtime"

	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
)

func itemURL(entry driveapi.NavEntry) string {
	if entry.WebViewLink != "" {
		return entry.WebViewLink
	}
	id := url.PathEscape(entry.ID)
	if entry.IsDir() {
		return "https://drive.google.com/drive/folders/" + id
	}
	return "https://drive.google.com/file/d/" + id + "/view"
}

// openBrowser starts the desktop browser and does not wait for it.
func openBrowser(link string) error {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	cmd := exec.Command(name, link)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
