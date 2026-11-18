package tui

import (
	"fmt"
	"strconv"
	"strings"
)

func (m *model) render() string {
	if m.help {
		return m.helpView()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", paint(styleTitle, m.heading()))
	m.writeRows(&b)
	if m.compose {
		fmt.Fprintf(&b, "\n%s\n", truncate("> "+m.draft, m.width))
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
	base := "slack"
	if label := m.presetLabel(); label != "" {
		base = "slack  " + label
	}
	if m.mode != modeMessages {
		return base
	}
	conv, ok := m.currentConv()
	if !ok {
		return base
	}
	return base + "  " + conv.Name
}

func (m *model) presetLabel() string {
	if m.preset < 0 || m.preset >= len(m.presets) {
		return ""
	}
	preset := m.presets[m.preset]
	if preset.Name == "" || preset.Name == preset.ID {
		return preset.ID
	}
	return preset.ID + "  " + preset.Name
}

func (m *model) writeRows(b *strings.Builder) {
	lines := m.lines()
	if len(lines) == 0 {
		fmt.Fprintf(b, "%s\n", m.emptyLine())
		return
	}
	start, end := m.window(len(lines))
	for i := start; i < end; i++ {
		prefix, style := "  ", ""
		if i == m.clamp() {
			prefix, style = "> ", styleActive
		}
		fmt.Fprintf(b, "%s\n", paint(style, truncate(prefix+lines[i], m.width)))
	}
}

func (m *model) lines() []string {
	switch m.mode {
	case modeWorkspaces:
		return m.workspaceLines()
	case modeConversations:
		return m.conversationLines()
	default:
		return m.messageLines()
	}
}

func (m *model) workspaceLines() []string {
	lines := make([]string, len(m.presets))
	for i, preset := range m.presets {
		label := preset.ID
		if preset.Name != "" && preset.Name != preset.ID {
			label = preset.ID + "  " + preset.Name
		}
		lines[i] = label
	}
	return lines
}

func (m *model) conversationLines() []string {
	lines := make([]string, len(m.conversations))
	for i, row := range m.conversations {
		lines[i] = row.label()
	}
	return lines
}

func (m *model) messageLines() []string {
	lines := make([]string, len(m.messages))
	for i, row := range m.messages {
		lines[i] = row.User + "  " + row.Text
	}
	return lines
}

func (row Conversation) label() string {
	unread := ""
	if row.Unread > 0 {
		unread = strconv.Itoa(row.Unread)
	}
	kind := row.Surface
	if kind == "channel" || kind == "private_channel" {
		kind = "#"
	}
	return fmt.Sprintf("%-5s %3s  %s", kind, unread, row.Name)
}

func (m *model) emptyLine() string {
	switch m.mode {
	case modeWorkspaces:
		return "no workspaces"
	case modeConversations:
		return "no conversations"
	default:
		return "no messages"
	}
}

func (m *model) helpView() string {
	lines := []string{
		paint(styleTitle, "slack"),
		"",
		"j/k    move",
		"h/l    workspace",
		"enter  open",
		"i      write",
		"r      reply",
		"m      mark read",
		"?      shortcuts",
		"q      quit",
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m *model) window(n int) (int, int) {
	rows := m.height - 6
	if rows < 1 {
		rows = 1
	}
	if n <= rows {
		return 0, n
	}
	start := 0
	cursor := m.clamp()
	if cursor >= rows {
		start = cursor - rows + 1
	}
	end := start + rows
	if end > n {
		end = n
	}
	return start, end
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
