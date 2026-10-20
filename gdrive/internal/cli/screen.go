package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ryangerardwilson/c3po/gdrive/internal/config"
	"github.com/ryangerardwilson/c3po/gdrive/internal/tui"
)

var errNoPresets = errors.New("no Drive presets. c3po gdrive auth adds one.")

// Screen asks for a preset and an action, then opens that Drive screen.
func Screen(stdout, stderr io.Writer) int {
	cfg, err := config.Load()
	if err != nil {
		return finish(stderr, err)
	}
	items := presetItems(cfg)
	if len(items) == 0 {
		return finish(stderr, errNoPresets)
	}
	choice, err := tui.Pick(items)
	if err != nil {
		return finish(stderr, err)
	}
	return runChoice(choice, stdout, stderr)
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

func runChoice(choice tui.Choice, stdout, stderr io.Writer) int {
	switch choice.Action {
	case "browse":
		return finish(stderr, runBrowse(choice.Preset, nil, stdout))
	case "upload":
		args := append([]string{"upload"}, strings.Fields(choice.Paths)...)
		return finish(stderr, runUploadPicker(choice.Preset, args, stdout))
	default:
		return finish(stderr, fmt.Errorf("unknown action %s", choice.Action))
	}
}

func finish(stderr io.Writer, err error) int {
	if err == nil || errors.Is(err, tui.ErrQuit) {
		return 0
	}
	fmt.Fprintln(stderr, err.Error())
	return 1
}
