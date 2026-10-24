package tui

import (
	"errors"
	"fmt"
	"strings"
)

// ErrQuit means the operator left the preset screen without choosing.
var ErrQuit = errors.New("quit")

// Item is one preset row.
type Item struct {
	ID    string
	Label string
}

type picker struct {
	presets []Item
	cursor  int
	preset  string
	quit    bool
	help    bool
}

func newPicker(presets []Item) *picker {
	return &picker{presets: presets}
}

// Pick opens the preset list and returns the chosen preset.
func Pick(presets []Item) (string, error) {
	term, err := openTerminal()
	if err != nil {
		return "", err
	}
	defer term.close()
	return chooseOn(term, presets)
}

func chooseOn(term screen, presets []Item) (string, error) {
	if len(presets) == 0 {
		return "", errors.New("no presets")
	}
	p := newPicker(presets)
	for {
		term.paint(p.frame())
		if p.finished() {
			break
		}
		key, err := term.readKey()
		if err != nil {
			return "", err
		}
		p.key(key)
	}
	if p.quit || p.preset == "" {
		return "", ErrQuit
	}
	return p.preset, nil
}

func (p *picker) finished() bool {
	return p.quit || p.preset != ""
}

func (p *picker) key(k string) {
	if p.help {
		switch k {
		case "?", "esc":
			p.help = false
		case "q", "ctrl+c":
			p.quit = true
		}
		return
	}
	switch k {
	case "?":
		p.help = true
	case "q", "esc", "ctrl+c":
		p.quit = true
	case "j", "down":
		p.move(1, len(p.presets))
	case "k", "up":
		p.move(-1, len(p.presets))
	case "l", "right", "enter":
		p.preset = p.presets[p.cursor].ID
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
	if p.help {
		lines := []string{
			paintStyle(styleTitle, "gdrive preset"),
			"",
			"j/k    move",
			"enter  select",
			"?      shortcuts",
			"q      quit",
		}
		return strings.Join(lines, "\n") + "\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", paintStyle(styleTitle, "gdrive preset"))
	for idx, item := range p.presets {
		prefix, style := "  ", ""
		if idx == p.cursor {
			prefix, style = "> ", styleActive
		}
		fmt.Fprintf(&b, "%s\n", paintStyle(style, prefix+item.Label))
	}
	return b.String()
}
