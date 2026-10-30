package app

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const cursorBlinkInterval = 500 * time.Millisecond

type cursorBlinkMsg time.Time

func renderTypedLine(raw, target, input string, width int, showCursor bool) string {
	if width <= 0 {
		return ""
	}
	inputRunes := []rune(input)
	targetRunes := []rune(target)
	var out strings.Builder
	displayPos := 0
	inputPos := 0
	cursorRendered := false
	for _, r := range raw {
		if displayPos >= width {
			break
		}
		if r == '\t' {
			for spaces := 0; spaces < 4 && displayPos < width; spaces++ {
				out.WriteRune(' ')
				displayPos++
			}
			continue
		}
		if showCursor && !cursorRendered && inputPos == len(inputRunes) {
			out.WriteString(statusStyle.Render("█"))
			displayPos++
			cursorRendered = true
			continue
		}
		ch := r
		style := mutedStyle
		if inputPos < len(inputRunes) {
			if inputPos < len(targetRunes) && inputRunes[inputPos] == targetRunes[inputPos] {
				ch = inputRunes[inputPos]
				style = statusStyle
			} else {
				ch = '█'
				style = errorStyle
			}
			inputPos++
		}
		out.WriteString(style.Render(string(ch)))
		displayPos++
	}
	if showCursor && !cursorRendered && inputPos == len(inputRunes) && displayPos < width {
		out.WriteString(statusStyle.Render("█"))
		displayPos++
	}
	for inputPos < len(inputRunes) && displayPos < width {
		out.WriteString(errorStyle.Render("█"))
		inputPos++
		displayPos++
	}
	return out.String()
}

func cursorBlinkCmd() tea.Cmd {
	return tea.Tick(cursorBlinkInterval, func(t time.Time) tea.Msg {
		return cursorBlinkMsg(t)
	})
}
