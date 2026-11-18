package tui

import (
	"fmt"
	"strings"
	"time"
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
	m.writeBanner(&b)
	fmt.Fprintf(&b, "\n")
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

func (m *model) writeBanner(b *strings.Builder) {
	wrote := false
	if line := m.syncedLine(); line != "" {
		fmt.Fprintf(b, "%s\n", paint(styleMuted, line))
		wrote = true
	}
	if text := m.notice(); text != "" {
		fmt.Fprintf(b, "%s\n", paint(styleError, text))
		wrote = true
	}
	if !wrote {
		fmt.Fprintf(b, "%s\n", paint(styleMuted, "no brief yet"))
	}
}

func (m *model) notice() string {
	if ws, ok := m.workspace(); ok {
		if text := clean(ws.Error); text != "" {
			return strings.ToUpper(text)
		}
	}
	if text := clean(m.board.Error); text != "" {
		return strings.ToUpper(text)
	}
	return ""
}

func (m *model) syncedLine() string {
	when := syncedAt(m.board.RefreshedAt)
	if when == "" {
		return ""
	}
	return "last synced at " + when
}

func syncedAt(value string) string {
	text := clean(value)
	if text == "" {
		return ""
	}
	when, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return text
	}
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		loc = time.FixedZone("IST", (5*60+30)*60)
	}
	return when.In(loc).Format("Mon, 2 Jan 2006, 3:04 PM MST")
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
		lines = append(lines, painted{text: "no items"})
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

func (m *model) helpView() string {
	lines := []string{
		paint(styleTitle, "slack"),
		"",
		"j/k    move",
		"h/l    workspace",
		"enter  open",
		"r      read",
		"s      sync",
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
