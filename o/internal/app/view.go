package app

import (
	"fmt"
	"strings"

	"github.com/ryangerardwilson/c3po/o/internal/fsnav"
	"github.com/ryangerardwilson/c3po/o/internal/style"
)

var (
	titleStyle    = style.NewStyle().Bold(true)
	mutedStyle    = style.NewStyle().Foreground(style.Color("241"))
	selectedStyle = style.NewStyle().Foreground(style.Color("229")).Background(style.Color("238"))
	statusStyle   = style.NewStyle().Foreground(style.Color("80"))
)

// Cascade phases of one row while a listing reveals.
const (
	phaseHidden = iota
	phaseStructure
	phaseFull
)

func (m model) textPreviewStats() (int, int) {
	maxLines := 0
	count := 0
	for _, row := range m.previewRows {
		if row.Entry.IsDir || !row.Entry.IsText {
			continue
		}
		count++
		maxLines = max(maxLines, row.Entry.Lines)
	}
	return maxLines, count
}

func textLabelStyle(entry fsnav.Entry, maxLines, textCount int) style.Style {
	if entry.IsDir || !entry.IsText {
		return style.NewStyle()
	}
	intensity := fsnav.TextIntensity(entry.Lines, maxLines, textCount)
	color := fmt.Sprintf("#%02X%02X00", intensity, intensity)
	return style.NewStyle().Foreground(style.Color(color))
}

func (m model) View() string {
	if m.showHelp {
		return padLines(m.renderHelp(), m.frameLines())
	}
	lines := []string{
		titleStyle.Render("o") + " " + mutedStyle.Render(truncateMiddle(m.currentDir, m.width-4)),
		m.promptLine(),
	}
	rows := m.rowLines()
	if len(m.entries) == 0 && len(rows) == 0 {
		rows = []string{mutedStyle.Render("No entries.")}
	}
	need := m.listingRows()
	if len(rows) > need {
		rows = rows[:need]
	}
	for len(rows) < need {
		rows = append(rows, "")
	}
	lines = append(lines, rows...)
	lines = append(lines, m.bottomLine())
	return strings.Join(lines, "\n")
}

func (m model) frameLines() int {
	if m.height < 1 {
		return 24
	}
	return m.height
}

func (m model) promptLine() string {
	switch {
	case m.filtering:
		text := "/ " + m.input
		return statusStyle.Render(blockLine(text, len([]rune(text)), m.width))
	default:
		if label := m.clipLabel(); label != "" {
			return mutedStyle.Render(label)
		}
		return ""
	}
}

func (m model) bottomLine() string {
	if m.commanding {
		return statusStyle.Render(m.commandLine())
	}
	return statusStyle.Render(m.status)
}

func (m model) commandLine() string {
	rs := []rune(m.input)
	at := clampAt(rs, m.inputAt)
	return blockLine(":"+string(rs), at+1, m.width)
}

// blockLine draws text with a reverse-video block at rune index at.
// An index at the end of text draws the block on a space.
func blockLine(text string, at, limit int) string {
	rs := []rune(text)
	if at < 0 {
		at = 0
	}
	if at > len(rs) {
		at = len(rs)
	}
	cell := " "
	before := text
	after := ""
	if at < len(rs) {
		cell = string(rs[at])
		before = string(rs[:at])
		after = string(rs[at+1:])
	}
	return fitCursor(before, cell, after, limit)
}

// fitCursor keeps the block on screen. A long line shows the tail when
// the cursor is at the end and the head when the cursor is at the start.
func fitCursor(before, cell, after string, limit int) string {
	left := []rune(before)
	mid := []rune(cell)
	if len(mid) == 0 {
		mid = []rune{' '}
	}
	right := []rune(after)
	full := make([]rune, 0, len(left)+len(mid)+len(right))
	full = append(full, left...)
	cursor := len(full)
	full = append(full, mid...)
	full = append(full, right...)
	start := 0
	end := len(full)
	if limit > 0 && len(full) > limit {
		if limit == 1 {
			return reverseCell(string(full[cursor]))
		}
		start = cursor - (limit - 1)
		if start < 0 {
			start = 0
		}
		end = start + limit
		if end > len(full) {
			end = len(full)
			start = end - limit
		}
	}
	window := full[start:end]
	cursorInWindow := cursor - start
	if start > 0 {
		window = window[1:]
		cursorInWindow--
		return "…" + paintBlock(window, cursorInWindow)
	}
	return paintBlock(window, cursorInWindow)
}

func paintBlock(window []rune, cursor int) string {
	var b strings.Builder
	for i, r := range window {
		if i == cursor {
			b.WriteString(reverseCell(string(r)))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func reverseCell(cell string) string {
	return "\x1b[7m" + cell + "\x1b[27m"
}

func padLines(view string, n int) string {
	lines := strings.Split(view, "\n")
	if n < 1 {
		n = 1
	}
	if len(lines) > n {
		lines = lines[:n]
	}
	for len(lines) < n {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m model) rowLines() []string {
	maxTextLines, textCount := m.textPreviewStats()
	start, end := m.visibleWindow()
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		row := m.previewRows[i]
		phase := m.rowPhase(i - start)
		if phase == phaseHidden {
			continue
		}
		if row.IsRoot {
			lines = append(lines, m.rootRow(row, phase, maxTextLines, textCount))
		} else {
			lines = append(lines, m.nestedRow(row, phase, maxTextLines, textCount))
		}
	}
	return lines
}

// rowPhase is how far the cascade has reached display row i: each row
// shows its structure first, then its label.
func (m model) rowPhase(displayIndex int) int {
	if !m.animationActive {
		return phaseFull
	}
	if m.animationStep == displayIndex*2+1 {
		return phaseStructure
	}
	if m.animationStep < (displayIndex+1)*2 {
		return phaseHidden
	}
	return phaseFull
}

func (m model) nestedRow(row fsnav.PreviewRow, phase, maxTextLines, textCount int) string {
	structure := "    " + row.Prefix
	if phase == phaseStructure {
		return mutedStyle.Render(structure)
	}
	label := row.Entry.Name + dirSuffix(row.Entry)
	labelStyle := textLabelStyle(row.Entry, maxTextLines, textCount)
	return mutedStyle.Render(structure) + labelStyle.Render(truncateMiddle(label, max(0, m.width-len(structure))))
}

func (m model) rootRow(row fsnav.PreviewRow, phase, maxTextLines, textCount int) string {
	entry := row.Entry
	mark := " "
	if m.marks[entry.Path] {
		mark = "*"
	}
	selected := row.RootIndex == m.selected
	prefix := "  " + mark + " "
	if selected {
		prefix = "> " + mark + " "
	}
	if phase == phaseStructure {
		return mutedStyle.Render(prefix)
	}
	label := truncateMiddle(entry.Name+dirSuffix(entry), m.width-3)
	labelStyle := textLabelStyle(entry, maxTextLines, textCount)
	if !selected && !m.rowInVisual(row.RootIndex) {
		return prefix + labelStyle.Render(label)
	}
	rowStyle := selectedStyle
	if entry.IsText {
		rowStyle = selectedStyle.Foreground(labelStyle.GetForeground())
	}
	return rowStyle.Render(prefix + label)
}

func dirSuffix(entry fsnav.Entry) string {
	if entry.IsDir {
		return "/"
	}
	return ""
}

func (m model) renderHelp() string {
	lines := []string{
		titleStyle.Render("o help"),
		"",
		"Listings use a staged cascade. Text brightens by relative line count.",
		"Structure is polled every 500ms and can be reloaded manually.",
		"j/k or arrows      move",
		"l or right         navigate/open/confirm without directory handoff",
		"enter              cd to the current listing directory in browse mode",
		"h or left          parent directory",
		"/                  enter or clear filter",
		": / :!             mv, rm, cp, touch, mkdir, shell",
		"!                  open the :! shell line",
		".                  show or hide dot entries",
		"= / -              increase or decrease nesting preview depth",
		"r                  reload the active listing immediately",
		"m                  mark/unmark current item",
		"v                  select consecutive items",
		"x / y              cut or copy the current item or the visual range",
		"p                  paste into this directory",
		"backspace/delete   delete the visual range",
		"dd                 delete the current item or the visual range",
		"n                  cd the invoking shell into this directory",
		"~                  home directory",
		"?                  toggle help",
		"q                  quit",
	}
	return strings.Join(lines, "\n")
}

func truncateMiddle(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit < 5 {
		return value[:limit]
	}
	half := (limit - 3) / 2
	return value[:half] + "..." + value[len(value)-(limit-3-half):]
}
