package app

import "strings"

func splitLessonLines(content string) []string {
	if content == "" {
		return []string{""}
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := strings.Split(content, "\n")
	if strings.HasSuffix(content, "\n") {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func (m model) typeStats() (int, int) {
	typed := 0
	total := 0
	for i, target := range m.typeTargets {
		if i < len(m.typeSkip) && m.typeSkip[i] {
			continue
		}
		total += len([]rune(target))
		input := ""
		if i < len(m.typeInputs) {
			input = m.typeInputs[i]
		}
		if i == m.typeLine && input == "" && m.typed != "" {
			input = m.typed
		}
		typed += len([]rune(input))
	}
	return typed, total
}

func (m model) typeVisibleRange(available int) (int, int) {
	total := len(m.typeLines)
	if total == 0 || available <= 0 {
		return 0, 0
	}
	if total <= available {
		return 0, total
	}
	offset := 0
	trigger := int(float64(available) * 0.6)
	if m.typeLine > trigger {
		offset = m.typeLine - trigger
	}
	linesBelow := total - 1 - m.typeLine
	if linesBelow <= 20 {
		desired := m.typeLine - int(float64(available)*0.3)
		if desired > offset {
			offset = desired
		}
	}
	offset = clamp(offset, 0, total-available)
	return offset, min(total, offset+available)
}

func (m model) renderTypeLine(index int) string {
	raw := ""
	if index < len(m.typeLines) {
		raw = m.typeLines[index]
	}
	target := ""
	if index < len(m.typeTargets) {
		target = m.typeTargets[index]
	}
	input := ""
	if index < len(m.typeInputs) {
		input = m.typeInputs[index]
	}
	if index == m.typeLine && input == "" && m.typed != "" {
		input = m.typed
	}
	showCursor := index == m.typeLine && !m.typeDone && m.cursorShown
	return renderTypedLine(raw, target, input, m.width, showCursor)
}
