package deport

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
	rows    []occupant
	cursor  int
	armed   bool
	status  string
	err     error
	width   int
	height  int
	load    func() ([]occupant, error)
	release func(port int) ([]proc, error)
	help    bool
}

func runScreen(e env) error {
	term, err := openTerminal()
	if err != nil {
		return err
	}
	defer term.close()
	m := &model{
		load: func() ([]occupant, error) { return occupied(e.procRoot) },
		release: func(port int) ([]proc, error) {
			return free(e.procRoot, port, e.killer, e.sleep)
		},
	}
	return m.loop(term)
}

func (m *model) loop(term screen) error {
	for {
		m.width, m.height = term.size()
		m.refresh()
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

func (m *model) refresh() {
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
	m.cursor = m.clamp()
}

// handleKey applies one key and reports whether the screen quits.
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
		m.armed = false
		m.status = ""
		return false
	}
	if m.armed {
		m.armed = false
		if key == "d" {
			m.freeSelected()
			return false
		}
		if key == "esc" {
			m.status = ""
			return false
		}
		m.status = ""
	} else if key == "d" {
		m.armed = true
		m.status = "d"
		return false
	}
	return m.motion(key)
}

func (m *model) motion(key string) bool {
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
	}
	m.cursor = m.clamp()
	return false
}

func (m *model) freeSelected() {
	if len(m.rows) == 0 {
		m.status = "no occupied external ports"
		return
	}
	row := m.rows[m.clamp()]
	_, err := m.release(row.port)
	if err != nil {
		m.err = err
		m.status = err.Error()
		return
	}
	m.err = nil
	m.status = fmt.Sprintf("%d is free", row.port)
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
	fmt.Fprintf(&b, "%s\n\n", paintStyle(styleTitle, "deport"))
	if len(m.rows) == 0 {
		b.WriteString("no occupied external ports\n")
	} else {
		start, end := m.window()
		for i := start; i < end; i++ {
			prefix, style := "  ", ""
			if i == m.clamp() {
				prefix, style = "> ", styleActive
			}
			line := truncate(prefix+formatOccupant(m.rows[i]), m.width)
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
		paintStyle(styleTitle, "deport"),
		"",
		"j/k    move",
		"dd     free the selected port",
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
