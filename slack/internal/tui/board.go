package tui

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
)

var (
	fetchBoard   = runShow
	openDeskItem = func(preset, id string) error { return runDesk("open", preset, id) }
	readDeskItem = func(preset, id string) error { return runDesk("read", preset, id) }
	syncDesk     = func() error { return runSync() }
)

func runSync() error {
	text, err := deskCombined("refresh")
	return syncResult(text, err)
}

func syncResult(text string, err error) error {
	if err != nil {
		return err
	}
	if strings.Contains(text, "refresh already running") {
		return errors.New("refresh already running")
	}
	return nil
}

type rawItem struct {
	MessageID    string `json:"message_id"`
	Conversation string `json:"conversation"`
	Sender       string `json:"sender"`
	Text         string `json:"text"`
	Date         string `json:"date"`
	Why          string `json:"why"`
}

type rawSpace struct {
	Preset string    `json:"preset"`
	Name   string    `json:"name"`
	Error  string    `json:"error"`
	DMs    []rawItem `json:"dms"`
	Tagged []rawItem `json:"tagged"`
	Aware  []rawItem `json:"aware"`
}

func runShow() (Board, error) {
	bin, err := exec.LookPath("rgw-slack-quickshell-plugin")
	if err != nil {
		return Board{}, errors.New("desk unavailable")
	}
	out, err := exec.Command(bin, "show").Output()
	if err != nil {
		return Board{}, errors.New("desk unavailable")
	}
	return decodeBoard(out)
}

func runDesk(action, preset, id string) error {
	_, err := deskCombined(action, preset, id)
	return err
}

func deskCombined(args ...string) (string, error) {
	bin, err := exec.LookPath("rgw-slack-quickshell-plugin")
	if err != nil {
		return "", errors.New("desk unavailable")
	}
	out, err := exec.Command(bin, args...).CombinedOutput()
	text := oneLine(string(out))
	if strings.Contains(text, "xoxp-") || strings.Contains(text, "xoxb-") {
		text = ""
	}
	if err != nil {
		if text == "" {
			text = "desk unavailable"
		}
		return "", errors.New(text)
	}
	return text, nil
}

func oneLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func decodeBoard(raw []byte) (Board, error) {
	var payload struct {
		RefreshedAt string     `json:"refreshed_at"`
		Since       string     `json:"since"`
		Error       string     `json:"error"`
		Workspaces  []rawSpace `json:"workspaces"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Board{}, errors.New("desk returned unreadable data")
	}
	board := Board{
		RefreshedAt: payload.RefreshedAt,
		Since:       payload.Since,
		Error:       payload.Error,
		Workspaces:  make([]Workspace, len(payload.Workspaces)),
	}
	for i, ws := range payload.Workspaces {
		board.Workspaces[i] = Workspace{
			Preset: ws.Preset,
			Name:   ws.Name,
			Error:  ws.Error,
			DMs:    itemsFrom(ws.DMs),
			Tagged: itemsFrom(ws.Tagged),
			Aware:  itemsFrom(ws.Aware),
		}
	}
	return board, nil
}

func itemsFrom(in []rawItem) []Item {
	out := make([]Item, len(in))
	for i, row := range in {
		out[i] = Item{
			MessageID:    row.MessageID,
			Conversation: row.Conversation,
			Sender:       row.Sender,
			Text:         row.Text,
			Date:         row.Date,
			Why:          row.Why,
		}
	}
	return out
}
