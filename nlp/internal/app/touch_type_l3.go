package app

import (
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateTypeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if !m.typeDone {
		m.cursorShown = true
	}
	if m.typeDone && (key == "n" || key == "N") {
		if m.lessonIndex < len(m.lessons)-1 {
			m.lessonIndex++
			m.scroll = 0
			m.startTyping()
			if !m.cursorBlinking {
				m.cursorBlinking = true
				return m, cursorBlinkCmd()
			}
			return m, nil
		}
		m.screen = screenCourses
		m.status = "ready"
		return m, nil
	}
	switch key {
	case "esc":
		m.screen = screenCourses
		m.status = "ready"
	case "ctrl+r":
		m.resetTyping()
	case "backspace", "ctrl+h":
		if !m.typeDone && !m.currentTypeLineIsSkip() {
			m.setCurrentTypeInput(trimLastRune(m.currentTypeInput()))
		}
	case "enter":
		if m.typeDone {
			return m, nil
		}
		if m.currentTypeLineIsSkip() {
			m.advanceTypeLine()
		} else if m.currentTypeInput() == m.currentTypeTarget() {
			m.advanceTypeLine()
		} else {
			m.status = "line does not match"
		}
	case "tab":
		if !m.typeDone && !m.currentTypeLineIsSkip() {
			input := m.currentTypeInput()
			target := m.currentTypeTarget()
			inputWidth := len([]rune(input))
			targetRunes := []rune(target)
			remaining := ""
			if inputWidth < len(targetRunes) {
				remaining = string(targetRunes[inputWidth:])
			}
			if strings.HasPrefix(remaining, "    ") {
				m.setCurrentTypeInput(input + "    ")
			}
		}
	default:
		if !m.typeDone && !m.currentTypeLineIsSkip() && len(msg.Runes) > 0 {
			input := m.currentTypeInput()
			target := m.currentTypeTarget()
			for _, r := range msg.Runes {
				if r < 32 || len([]rune(input)) >= len([]rune(target)) {
					continue
				}
				input += string(r)
			}
			m.setCurrentTypeInput(input)
		}
	}
	if m.screen == screenType && m.allTypeLinesComplete() {
		m.typeDone = true
		m.cursorShown = true
		m.status = "lesson complete"
	}
	return m, nil
}

func (m *model) startTyping() {
	lines := splitLessonLines(m.currentLesson().Content)
	targets := make([]string, len(lines))
	skip := make([]bool, len(lines))
	for i, line := range lines {
		targets[i] = strings.ReplaceAll(line, "\t", "")
		trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
		skip[i] = strings.HasPrefix(trimmed, "#!") || strings.HasPrefix(trimmed, "//!") || strings.HasPrefix(trimmed, "--!")
	}
	m.typeLines = lines
	m.typeTargets = targets
	m.typeSkip = skip
	m.typeInputs = make([]string, len(lines))
	m.typeLine = 0
	m.typed = ""
	m.typeDone = false
	m.cursorShown = true
	m.screen = screenType
	m.status = "type mode"
}

func (m model) currentTypeTarget() string {
	if len(m.typeTargets) == 0 {
		return ""
	}
	return m.typeTargets[clamp(m.typeLine, 0, len(m.typeTargets)-1)]
}

func (m model) renderType() string {
	contentHeight := max(0, m.height-5)
	offset, end := m.typeVisibleRange(contentHeight)
	var out strings.Builder
	out.WriteString(titleStyle.Render(truncateMiddle(m.sequenceName, m.width)))
	out.WriteString("\n")
	out.WriteString(statusStyle.Render(truncateMiddle("TOUCH_TYPE_MODE: "+m.currentLesson().Name, m.width)))
	out.WriteString("\n")
	out.WriteString("\n")
	for i := offset; i < end; i++ {
		out.WriteString(m.renderTypeLine(i))
		out.WriteString("\n")
	}
	for i := end - offset; i < contentHeight; i++ {
		out.WriteString("\n")
	}
	typed, total := m.typeStats()
	stats := fmt.Sprintf("Typed %d/%d chars", typed, total)
	if len(m.typeLines) > contentHeight && contentHeight > 0 {
		stats += fmt.Sprintf("  [%d-%d/%d]", offset+1, end, len(m.typeLines))
	}
	out.WriteString(statusStyle.Render(truncateMiddle(stats, m.width)))
	return out.String()
}

func (m *model) resetTyping() {
	if len(m.typeInputs) != len(m.typeLines) {
		m.typeInputs = make([]string, len(m.typeLines))
	} else {
		for i := range m.typeInputs {
			m.typeInputs[i] = ""
		}
	}
	m.typed = ""
	m.typeLine = 0
	m.typeDone = false
	m.cursorShown = true
	m.status = "typing reset"
}

func (m model) currentTypeInput() string {
	if len(m.typeInputs) == 0 {
		return m.typed
	}
	index := clamp(m.typeLine, 0, len(m.typeInputs)-1)
	if m.typeInputs[index] == "" && m.typed != "" {
		return m.typed
	}
	return m.typeInputs[index]
}

func (m *model) setCurrentTypeInput(value string) {
	if len(m.typeInputs) == 0 {
		m.typed = value
		return
	}
	index := clamp(m.typeLine, 0, len(m.typeInputs)-1)
	m.typeInputs[index] = value
	m.typed = value
}

func (m model) currentTypeLineIsSkip() bool {
	if len(m.typeSkip) == 0 {
		return false
	}
	return m.typeSkip[clamp(m.typeLine, 0, len(m.typeSkip)-1)]
}

func (m *model) advanceTypeLine() {
	if len(m.typeLines) == 0 {
		m.typeDone = true
		m.cursorShown = true
		m.status = "lesson complete"
		return
	}
	if m.typeLine < len(m.typeLines)-1 {
		m.typeLine++
		m.typed = ""
		m.cursorShown = true
		m.status = "next line"
		return
	}
	m.typeDone = true
	m.cursorShown = true
	m.status = "lesson complete"
}

func (m model) allTypeLinesComplete() bool {
	if len(m.typeTargets) == 0 {
		return false
	}
	for i, target := range m.typeTargets {
		if i < len(m.typeSkip) && m.typeSkip[i] {
			continue
		}
		input := ""
		if i < len(m.typeInputs) {
			input = m.typeInputs[i]
		}
		if i == m.typeLine && input == "" && m.typed != "" {
			input = m.typed
		}
		if input != target {
			return false
		}
	}
	return true
}
