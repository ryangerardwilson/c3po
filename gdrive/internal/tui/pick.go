package tui

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ErrQuit means the operator left the preset screen without choosing.
var ErrQuit = errors.New("quit")

// Item is one row in the preset or action list.
type Item struct {
	ID    string
	Label string
}

// Choice is the preset, the action, and the upload paths.
type Choice struct {
	Preset string
	Action string
	Paths  string
}

const (
	stagePreset = "preset"
	stageAction = "action"
	stagePaths  = "paths"
)

var driveActions = []Item{
	{ID: "browse", Label: "browse"},
	{ID: "upload", Label: "upload"},
}

type picker struct {
	presets []Item
	stage   string
	cursor  int
	preset  string
	action  string
	draft   string
	paths   string
	status  string
	quit    bool
}

func newPicker(presets []Item) *picker {
	return &picker{presets: presets, stage: stagePreset}
}

// Pick opens the preset and action screen.
func Pick(presets []Item) (Choice, error) {
	term, err := openTerminal()
	if err != nil {
		return Choice{}, err
	}
	defer term.close()
	return chooseOn(term, presets)
}

func chooseOn(term screen, presets []Item) (Choice, error) {
	if len(presets) == 0 {
		return Choice{}, errors.New("no presets")
	}
	p := newPicker(presets)
	for {
		term.paint(p.frame())
		if p.finished() {
			break
		}
		key, err := term.readKey()
		if err != nil {
			return Choice{}, err
		}
		p.key(key)
	}
	if p.quit || p.action == "" {
		return Choice{}, ErrQuit
	}
	return p.choice(), nil
}

func (p *picker) choice() Choice {
	return Choice{Preset: p.preset, Action: p.action, Paths: p.paths}
}

func (p *picker) finished() bool {
	if p.quit {
		return true
	}
	if p.action == "browse" {
		return true
	}
	return p.action == "upload" && p.paths != ""
}

func (p *picker) key(k string) {
	switch p.stage {
	case stageAction:
		p.keyAction(k)
	case stagePaths:
		p.keyPaths(k)
	default:
		p.keyPreset(k)
	}
}

func (p *picker) keyPreset(k string) {
	switch k {
	case "q", "esc", "ctrl+c":
		p.quit = true
	case "j", "down":
		p.move(1, len(p.presets))
	case "k", "up":
		p.move(-1, len(p.presets))
	case "l", "right", "enter":
		p.preset = p.presets[p.cursor].ID
		p.cursor = 0
		p.stage = stageAction
		p.status = ""
	}
}

func (p *picker) keyAction(k string) {
	switch k {
	case "q", "ctrl+c":
		p.quit = true
	case "h", "left", "backspace", "esc":
		p.cursor = indexOf(p.presets, p.preset)
		p.stage = stagePreset
		p.status = ""
	case "j", "down":
		p.move(1, len(driveActions))
	case "k", "up":
		p.move(-1, len(driveActions))
	case "l", "right", "enter":
		p.chooseAction()
	}
}

func (p *picker) chooseAction() {
	p.action = driveActions[p.cursor].ID
	if p.action != "upload" {
		return
	}
	p.stage = stagePaths
	p.draft = ""
	p.status = ""
}

func (p *picker) keyPaths(k string) {
	switch k {
	case "ctrl+c":
		p.quit = true
	case "esc":
		p.stage = stageAction
		p.action = ""
		p.status = ""
	case "enter":
		p.acceptPaths()
	case "backspace":
		p.draft = dropLast(p.draft)
	default:
		p.typeKey(k)
	}
}

func (p *picker) acceptPaths() {
	if strings.TrimSpace(p.draft) == "" {
		p.status = "enter one or more local paths"
		return
	}
	p.paths = strings.TrimSpace(p.draft)
	p.status = ""
}

func (p *picker) typeKey(k string) {
	if len(k) == 1 && k[0] >= 32 {
		p.draft += k
	}
}

func (p *picker) move(delta, n int) {
	if n == 0 {
		return
	}
	next := p.cursor + delta
	if next < 0 {
		next = 0
	}
	if next >= n {
		next = n - 1
	}
	p.cursor = next
}

func (p *picker) frame() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", paintStyle(styleTitle, p.heading()))
	fmt.Fprintf(&b, "%s\n\n", paintStyle(styleMuted, p.hint()))
	if p.stage == stagePaths {
		b.WriteString(p.draft)
		b.WriteByte('\n')
	} else {
		p.writeItems(&b)
	}
	if p.status != "" {
		fmt.Fprintf(&b, "\n%s\n", paintStyle(styleError, p.status))
	}
	return b.String()
}

func (p *picker) heading() string {
	switch p.stage {
	case stageAction:
		return "gdrive action  preset " + p.preset
	case stagePaths:
		return "gdrive paths  preset " + p.preset
	default:
		return "gdrive preset"
	}
}

func (p *picker) hint() string {
	switch p.stage {
	case stageAction:
		return "j/k move  enter select  h back  q quit"
	case stagePaths:
		return "type local paths  enter continue  esc back"
	default:
		return "j/k move  enter select  q quit"
	}
}

func (p *picker) writeItems(b *strings.Builder) {
	items := p.presets
	if p.stage == stageAction {
		items = driveActions
	}
	for idx, item := range items {
		prefix, style := "  ", ""
		if idx == p.cursor {
			prefix, style = "> ", styleActive
		}
		fmt.Fprintf(b, "%s\n", paintStyle(style, prefix+item.Label))
	}
}

func indexOf(items []Item, id string) int {
	for idx, item := range items {
		if item.ID == id {
			return idx
		}
	}
	return 0
}

func dropLast(value string) string {
	if value == "" {
		return ""
	}
	_, size := utf8.DecodeLastRuneInString(value)
	return value[:len(value)-size]
}
