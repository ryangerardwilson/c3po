package app

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/ryangerardwilson/c3po/o/internal/fsnav"
	"github.com/ryangerardwilson/c3po/o/internal/tui"
)

// keyHandler applies one browse key. Any listing rebuild it makes is
// picked up by withAnimation.
type keyHandler func(m model) (model, tui.Cmd)

var browseKeys = map[string]keyHandler{
	"r":      keyReload,
	"up":     keyUp,
	"k":      keyUp,
	"down":   keyDown,
	"j":      keyDown,
	"ctrl+k": keyJumpUp,
	"ctrl+j": keyJumpDown,
	"left":   keyParent,
	"h":      keyParent,
	"right":  model.activate,
	"l":      model.activate,
	"enter":  model.accept,
	"/":      keyFilter,
	"!":      keyCommand,
	"=":      keyDeeper,
	"+":      keyDeeper,
	"-":      keyShallower,
	"_":      keyShallower,
	".":      keyDots,
	"m":      keyMark,
	"v":      keyVisual,
	"x":      keyCut,
	"y":      keyCopy,
	"p":      keyPaste,
	"n":      keyShellCD,
	"~":      keyHome,
}

func (m model) updateKey(msg tui.KeyMsg) (tui.Model, tui.Cmd) {
	key := msg.String()
	animationGeneration := m.animationGeneration
	watchGeneration := m.watchGeneration
	if m.commanding {
		return m.withAnimation(m.commandKey(msg, key), nil, animationGeneration, watchGeneration)
	}
	if m.filtering {
		return m.withAnimation(m.filterKey(msg, key), nil, animationGeneration, watchGeneration)
	}
	if m.deleteArmed {
		m.deleteArmed = false
		if key == "d" {
			m.deleteCurrent()
			return m.withAnimation(m, nil, animationGeneration, watchGeneration)
		}
		if key == "esc" {
			m.visual = false
			m.status = ""
			return m, nil
		}
		m.status = ""
	}
	if key == "ctrl+c" || key == "q" {
		return m, tui.Quit
	}
	if key == "?" || m.showHelp {
		return m.helpKey(key), nil
	}
	if msg.Type == tui.KeyDelete {
		if !m.visual {
			return m, nil
		}
		m.deleteCurrent()
		return m.withAnimation(m, nil, animationGeneration, watchGeneration)
	}
	if key == "esc" && m.visual {
		m.visual = false
		m.status = ""
		return m, nil
	}
	handler, ok := browseKeys[key]
	if !ok {
		if key == "d" {
			m.deleteArmed = true
			m.status = "d"
		}
		return m, nil
	}
	next, cmd := handler(m)
	return m.withAnimation(next, cmd, animationGeneration, watchGeneration)
}

// helpKey handles keys while the help overlay is shown, and ? anywhere.
func (m model) helpKey(key string) model {
	switch {
	case key == "?":
		m.showHelp = !m.showHelp
	case key == "esc" || key == "h":
		m.showHelp = false
	}
	return m
}

func (m model) filterKey(msg tui.KeyMsg, key string) model {
	switch key {
	case "esc", "ctrl+c":
		m.filtering = false
		m.input = ""
		m.status = "filter cancelled"
	case "enter":
		m.filter = strings.TrimSpace(m.input)
		m.filtering = false
		m.reload()
		m.status = "filter: " + m.filter
	case "backspace", "ctrl+h":
		m.input = trimLastRune(m.input)
	default:
		if len(msg.Runes) > 0 {
			m.input += string(msg.Runes)
		}
	}
	return m
}

func keyReload(m model) (model, tui.Cmd) {
	m.reloadPreservingSelection()
	m.status = "reloaded"
	return m, nil
}

func keyUp(m model) (model, tui.Cmd) {
	m.move(-1)
	return m, nil
}

func keyDown(m model) (model, tui.Cmd) {
	m.move(1)
	return m, nil
}

func keyJumpUp(m model) (model, tui.Cmd) {
	m.move(-max(1, len(m.entries)/10))
	return m, nil
}

func keyJumpDown(m model) (model, tui.Cmd) {
	m.move(max(1, len(m.entries)/10))
	return m, nil
}

func keyParent(m model) (model, tui.Cmd) {
	m.parent()
	return m, nil
}

func keyCommand(m model) (model, tui.Cmd) {
	m.filtering = false
	m.commanding = true
	m.input = ""
	m.status = ""
	return m, nil
}

func (m model) commandKey(msg tui.KeyMsg, key string) model {
	switch key {
	case "esc", "ctrl+c":
		m.commanding = false
		m.input = ""
		m.status = "command cancelled"
	case "enter":
		line := strings.TrimSpace(m.input)
		m.commanding = false
		m.input = ""
		status, path, err := applyFileCommand(m.currentDir, line)
		if err != nil {
			m.status = err.Error()
			return m
		}
		m.reload()
		m.selectPath(path)
		m.status = status
	case "backspace", "ctrl+h":
		m.input = trimLastRune(m.input)
	default:
		if len(msg.Runes) > 0 {
			m.input += string(msg.Runes)
		}
	}
	return m
}

func keyFilter(m model) (model, tui.Cmd) {
	if m.filter != "" {
		m.filter = ""
		m.reload()
		m.status = "filter cleared"
		return m, nil
	}
	m.filtering = true
	m.input = ""
	m.status = "filter"
	return m, nil
}

func keyDeeper(m model) (model, tui.Cmd) {
	if m.previewDepth >= fsnav.MaxPreviewDepth {
		m.status = "preview depth: maximum"
		return m, nil
	}
	m.previewDepth++
	m.reload()
	m.status = fmt.Sprintf("preview depth: %d", m.previewDepth)
	return m, nil
}

func keyShallower(m model) (model, tui.Cmd) {
	if m.previewDepth <= 0 {
		m.status = "preview depth: minimum"
		return m, nil
	}
	m.previewDepth--
	m.reload()
	m.status = fmt.Sprintf("preview depth: %d", m.previewDepth)
	return m, nil
}

func keyDots(m model) (model, tui.Cmd) {
	m.showHidden = !m.showHidden
	m.selected = 0
	m.reload()
	m.status = "dot entries hidden"
	if m.showHidden {
		m.status = "dot entries shown"
	}
	return m, nil
}

func keyMark(m model) (model, tui.Cmd) {
	m.toggleMark()
	return m, nil
}

func keyVisual(m model) (model, tui.Cmd) {
	if m.visual {
		m.visual = false
		m.status = ""
		return m, nil
	}
	if len(m.entries) == 0 || m.selected < 0 || m.selected >= len(m.entries) {
		m.status = "nothing to select"
		return m, nil
	}
	m.visual = true
	m.visualAt = m.selected
	m.status = m.visualStatus()
	return m, nil
}

func keyShellCD(m model) (model, tui.Cmd) {
	if m.cfg.Mode != ModeBrowse {
		return m, nil
	}
	m.result.ShellCD = m.currentDir
	return m, tui.Quit
}

func keyCut(m model) (model, tui.Cmd) {
	m.stageClip(true)
	return m, nil
}

func keyCopy(m model) (model, tui.Cmd) {
	m.stageClip(false)
	return m, nil
}

func keyPaste(m model) (model, tui.Cmd) {
	m.pasteClip()
	return m, nil
}

func keyHome(m model) (model, tui.Cmd) {
	if home, err := os.UserHomeDir(); err == nil {
		m.currentDir = home
		m.reload()
	}
	return m, nil
}

func trimLastRune(value string) string {
	if value == "" {
		return value
	}
	_, size := utf8.DecodeLastRuneInString(value)
	return value[:len(value)-size]
}
