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

// Screen asks for a preset, then browses that Drive.
func Screen(stdout, stderr io.Writer) int {
	cfg, err := config.Load()
	if err != nil {
		return finish(stderr, err)
	}
	items := presetItems(cfg)
	if len(items) == 0 {
		return finish(stderr, errNoPresets)
	}
	preset, err := tui.Pick(items)
	if err != nil {
		return finish(stderr, err)
	}
	return finish(stderr, runBrowse(preset, nil, stdout))
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
