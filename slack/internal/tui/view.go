package tui

import (
	"fmt"
	"strings"
)

type painted struct {
	text   string
	active bool
}

func (m *model) render() string {
	if m.help {
		return m.helpView()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", m.heading())
	fmt.Fprintf(&b, "%s\n\n", m.noticeLine())
	for _, line := range m.window(m.table()) {
		fmt.Fprintf(&b, "%s\n", line.text)
	}
	if m.status != "" {
		style := styleMuted
		if m.err != nil {
			style = styleError
		}
		fmt.Fprintf(&b, "\n%s\n", paint(style, m.status))
	}
	return b.String()
}

func (m *model) heading() string {
	var b strings.Builder
	b.WriteString(paint(styleTitle, "slack"))
	for i, ws := range m.board.Workspaces {
		style := styleMuted
		if i == m.ws {
			style = styleActive
		}
		b.WriteString("  ")
		b.WriteString(paint(style, workspaceName(ws)))
	}
	return b.String()
}

func workspaceName(ws Workspace) string {
	if name := clean(ws.Name); name != "" {
		return name
	}
	return clean(ws.Preset)
}

func (m *model) notice() (string, bool) {
	if ws, ok := m.workspace(); ok {
		if text := clean(ws.Error); text != "" {
			return strings.ToUpper(text), true
		}
	}
	if text := clean(m.board.Error); text != "" {
		return strings.ToUpper(text), true
	}
	if text := clean(m.board.RefreshedAt); text != "" {
		return text, false
	}
	return "no brief yet", false
}

func (m *model) noticeLine() string {
	text, bad := m.notice()
	if bad {
		return paint(styleError, text)
	}
	return paint(styleMuted, text)
}

func (m *model) table() []painted {
	ws, ok := m.workspace()
	if !ok {
		return []painted{{text: "no brief yet"}}
	}
	if m.count() == 0 {
		lines := make([]painted, 0, 4)
		for _, part := range ws.buckets() {
			lines = append(lines, painted{text: paint(styleTitle, part.label+"  0")})
		}
		lines = append(lines, painted{text: m.emptyNote()})
		return lines
	}
	lines := []painted{{text: paint(styleMuted, columnHead())}}
	selected := m.clamp()
	index := 0
	for _, part := range ws.buckets() {
		lines = append(lines, painted{text: paint(styleTitle, fmt.Sprintf("%s  %d", part.label, len(part.items)))})
		for _, item := range part.items {
			prefix, style := "  ", ""
			active := index == selected
			if active {
				prefix, style = "> ", styleActive
			}
			lines = append(lines, painted{
				text:   paint(style, truncate(prefix+itemLine(item), m.width)),
				active: active,
			})
			index++
		}
	}
	return lines
}

func (m *model) emptyNote() string {
	if since := clean(m.board.Since); since != "" {
		return "watching after  " + since
	}
	return "no items"
}

func (m *model) helpView() string {
	lines := []string{
		paint(styleTitle, "slack"),
		"",
		"j/k    move",
		"h/l    workspace",
		"enter  open",
		"r      read",
		"?      shortcuts",
		"q      quit",
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m *model) window(lines []painted) []painted {
	rows := m.height - 6
	if rows < 1 {
		rows = 1
	}
	if len(lines) <= rows {
		return lines
	}
	at := 0
	for i, line := range lines {
		if line.active {
			at = i
			break
		}
	}
	start := 0
	if at >= rows {
		start = at - rows + 1
	}
	end := start + rows
	if end > len(lines) {
		end = len(lines)
	}
	return lines[start:end]
}

func paint(style, text string) string {
	if style == "" || text == "" {
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
