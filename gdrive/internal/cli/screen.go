package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ryangerardwilson/c3po/gdrive/internal/config"
	"github.com/ryangerardwilson/c3po/gdrive/internal/tui"
	"github.com/ryangerardwilson/c3po/internal/tuisession"
)

var errNoPresets = errors.New("no Drive presets. c3po gdrive auth adds one.")

// Screen asks for a preset, then browses that Drive.
func Screen(stdout, stderr io.Writer) int {
	return screen(stdout, stderr, "")
}

// ScreenResume opens the saved preset and folder.
func ScreenResume(id string, stdout, stderr io.Writer) int {
	return screen(stdout, stderr, id)
}

func screen(stdout, stderr io.Writer, id string) int {
	sess, err := tuisession.Open("gdrive", id)
	if err != nil {
		return finish(stderr, err)
	}
	defer sess.Close()
	cfg, err := config.Load()
	if err != nil {
		return finish(stderr, err)
	}
	items := presetItems(cfg)
	if len(items) == 0 {
		return finish(stderr, errNoPresets)
	}
	preset := ""
	if body, ok := sess.Restored(); ok && len(body.Args) > 0 && knownPreset(items, body.Args[0]) {
		preset = body.Args[0]
	}
	if preset == "" {
		preset, err = tui.Pick(items)
		if err != nil {
			return finish(stderr, err)
		}
	}
	if err := sess.Note(tuisession.Body{Args: []string{preset}}); err != nil {
		return finish(stderr, err)
	}
	return finish(stderr, runBrowse(preset, sess))
}

func knownPreset(items []tui.Item, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func presetItems(cfg *config.AppConfig) []tui.Item {
	keys := sortedAccountKeys(cfg.Accounts)
	items := make([]tui.Item, 0, len(keys))
	for _, key := range keys {
		label := key
		if email := strings.TrimSpace(cfg.Accounts[key].Email); email != "" {
			label = key + "  " + email
		}
		items = append(items, tui.Item{ID: key, Label: label})
	}
	return items
}

func finish(stderr io.Writer, err error) int {
	if err == nil || errors.Is(err, tui.ErrQuit) {
		return 0
	}
	fmt.Fprintln(stderr, err.Error())
	return 1
}
