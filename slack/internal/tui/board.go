package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// Each desk call runs on the screen's worker with its own bound. A refresh
// may spend the desk's whole Slack and Grok budget.
const (
	showWait    = 10 * time.Second
	readWait    = 10 * time.Second
	openWait    = 30 * time.Second
	refreshWait = 5 * time.Minute
)

var (
	fetchBoard   = runShow
	openDeskItem = func(preset, id string) error { return runDesk(openWait, "open", preset, id) }
	readDeskItem = runRead
	syncDesk     = func() error { return runSync() }
)

func runSync() error {
	text, err := deskCombined(refreshWait, "refresh")
	return syncResult(text, err)
}

// runRead marks one row read and returns the desk the read printed. A desk
// that prints nothing after a read is asked to show the desk.
func runRead(preset, id string) (Board, error) {
	out, err := deskOutput(readWait, "read", preset, id)
	if err != nil {
		return Board{}, err
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return fetchBoard()
	}
	return decodeBoard(out)
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
	ctx, cancel := context.WithTimeout(context.Background(), showWait)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "show").Output()
	if err != nil {
		return Board{}, errors.New("desk unavailable")
	}
	return decodeBoard(out)
}

func runDesk(wait time.Duration, action, preset, id string) error {
	_, err := deskCombined(wait, action, preset, id)
	return err
}

func deskCombined(wait time.Duration, args ...string) (string, error) {
	bin, err := exec.LookPath("rgw-slack-quickshell-plugin")
	if err != nil {
		return "", errors.New("desk unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, args...).CombinedOutput()
	if ctx.Err() != nil {
		return "", errors.New("desk timed out")
	}
	text := safeText(string(out))
	if err != nil {
		if text == "" {
			text = "desk unavailable"
		}
		return "", errors.New(text)
	}
	return text, nil
}

// deskOutput runs one desk command and returns stdout. A failure carries
// stderr as one line.
func deskOutput(wait time.Duration, args ...string) ([]byte, error) {
	bin, err := exec.LookPath("rgw-slack-quickshell-plugin")
	if err != nil {
		return nil, errors.New("desk unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, errors.New("desk timed out")
	}
	if err != nil {
		text := safeText(stderr.String())
		if text == "" {
			text = "desk unavailable"
		}
		return nil, errors.New(text)
	}
	return out, nil
}

// safeText is desk output as one line, or nothing when it holds a token.
func safeText(raw string) string {
	text := oneLine(raw)
	if strings.Contains(text, "xoxp-") || strings.Contains(text, "xoxb-") {
		return ""
	}
	return text
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
