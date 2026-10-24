package looptab

import (
	"fmt"
	"strings"
)

const (
	styleTitle  = "\x1b[1m"
	styleActive = "\x1b[1;92m"
	styleMuted  = "\x1b[90m"
	styleError  = "\x1b[91m"
	styleReset  = "\x1b[0m"
)

// screen is the terminal the list draws on and reads keys from.
type screen interface {
	size() (width, height int)
	paint(frame string)
	readKey() (string, error)
}

type model struct {
	rows   []job
	cursor int
	status string
	err    error
	help   bool
	width  int
	height int
	load   func() ([]job, error)
	run    func(job) error
}

func (m *model) loop(term screen) error {
	m.reload()
	for {
		m.width, m.height = term.size()
		term.paint(m.view())
		key, err := term.readKey()
		if err != nil {
			return err
		}
		if m.handleKey(key) {
			return nil
		}
	}
}

func (m *model) reload() {
	if m.load == nil {
		return
	}
	rows, err := m.load()
	if err != nil {
		m.err = err
		m.status = err.Error()
		m.rows = nil
		m.cursor = 0
		return
	}
	m.rows = rows
	m.err = nil
	m.cursor = m.clamp()
}

func (m *model) handleKey(key string) bool {
	if m.help {
		switch key {
		case "?", "esc":
			m.help = false
		case "ctrl+c", "q":
			return true
		}
		return false
	}
	if key == "?" {
		m.help = true
		m.status = ""
		return false
	}
	switch key {
	case "ctrl+c", "q", "esc":
		return true
	case "j", "down":
		m.cursor++
	case "k", "up":
		m.cursor--
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = len(m.rows) - 1
	case "enter":
		m.trigger()
	}
	m.cursor = m.clamp()
	return false
}

func (m *model) trigger() {
	if len(m.rows) == 0 {
		if m.err == nil {
			m.status = "no jobs"
		}
		return
	}
	if m.run == nil {
		return
	}
	row := m.rows[m.clamp()]
	err := m.run(row)
	m.reload()
	if err != nil {
		m.err = err
		m.status = row.ID + " failed"
		return
	}
	m.err = nil
	m.status = "ran " + row.ID
}

func (m *model) clamp() int {
	if len(m.rows) == 0 || m.cursor < 0 {
		return 0
	}
	if m.cursor >= len(m.rows) {
		return len(m.rows) - 1
	}
	return m.cursor
}

func (m *model) view() string {
	if m.help {
		return m.helpView()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", paintStyle(styleTitle, "looptab"))
	if len(m.rows) == 0 {
		if m.err == nil {
			b.WriteString("no jobs\n")
		}
	} else {
		start, end := m.window()
		for i := start; i < end; i++ {
			prefix, style := "  ", ""
			if i == m.clamp() {
				prefix, style = "> ", styleActive
			}
			line := truncate(prefix+m.rows[i].label(), m.width)
			b.WriteString(paintStyle(style, line))
			b.WriteByte('\n')
		}
	}
	if m.status != "" {
		style := styleMuted
		if m.err != nil {
			style = styleError
		}
		fmt.Fprintf(&b, "\n%s\n", paintStyle(style, m.status))
	}
	return b.String()
}

func (m *model) helpView() string {
	lines := []string{
		paintStyle(styleTitle, "looptab"),
		"",
		"j/k    move",
		"enter  run now",
		"?      shortcuts",
		"q      quit",
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m *model) window() (int, int) {
	rows := m.height - 3
	if m.height <= 0 || rows < 1 || len(m.rows) <= rows {
		return 0, len(m.rows)
	}
	start := 0
	if m.cursor >= rows {
		start = m.cursor - rows + 1
	}
	end := start + rows
	if end > len(m.rows) {
		end = len(m.rows)
	}
	return start, end
}

func paintStyle(style, text string) string {
	if style == "" {
		return text
	}
	return style + text + styleReset
}

func truncate(value string, width int) string {
	if width <= 4 || len(value) <= width {
		return value
	}
	return value[:width-3] + "..."
}
