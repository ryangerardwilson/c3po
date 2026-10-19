// Package style renders bold text and terminal colors for the navigator.
// It covers the small part of a styling library the screen uses: a
// foreground, a background, and bold.
package style

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// TerminalColor is a color a Style can paint with.
type TerminalColor interface {
	sequence(fg bool) string
}

// NoColor leaves the terminal default color in place.
type NoColor struct{}

func (NoColor) sequence(bool) string { return "" }

// Color is an ANSI 256 index such as "241" or a hex color such as "#B4B400".
type Color string

func (c Color) sequence(fg bool) string {
	base := "38"
	if !fg {
		base = "48"
	}
	value := string(c)
	if strings.HasPrefix(value, "#") {
		return hexSequence(base, value)
	}
	if n, err := strconv.Atoi(value); err == nil && n >= 0 && n <= 255 {
		return fmt.Sprintf("%s;5;%d", base, n)
	}
	return ""
}

func hexSequence(base, value string) string {
	r, g, b, ok := parseHex(value)
	if !ok {
		return ""
	}
	if trueColor() {
		return fmt.Sprintf("%s;2;%d;%d;%d", base, r, g, b)
	}
	return fmt.Sprintf("%s;5;%d", base, cubeIndex(r, g, b))
}

func parseHex(value string) (int, int, int, bool) {
	if len(value) != 7 {
		return 0, 0, 0, false
	}
	n, err := strconv.ParseUint(value[1:], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(n >> 16 & 0xff), int(n >> 8 & 0xff), int(n & 0xff), true
}

// cubeIndex maps a true color to the nearest entry of the 6x6x6 cube.
func cubeIndex(r, g, b int) int {
	return 16 + 36*cubeLevel(r) + 6*cubeLevel(g) + cubeLevel(b)
}

func cubeLevel(v int) int {
	if v < 48 {
		return 0
	}
	if v < 115 {
		return 1
	}
	return (v - 35) / 40
}

var trueColorTerms = map[string]bool{
	"alacritty":     true,
	"contour":       true,
	"rio":           true,
	"wezterm":       true,
	"xterm-ghostty": true,
	"xterm-kitty":   true,
	"foot":          true,
	"foot-extra":    true,
}

func trueColor() bool {
	colorterm := strings.ToLower(os.Getenv("COLORTERM"))
	if colorterm == "truecolor" || colorterm == "24bit" {
		return true
	}
	return trueColorTerms[os.Getenv("TERM")]
}

func colorsEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return os.Getenv("TERM") != "dumb"
}

// Style is an immutable set of text attributes.
type Style struct {
	bold bool
	fg   TerminalColor
	bg   TerminalColor
}

// NewStyle is a style with no attributes.
func NewStyle() Style { return Style{} }

// Bold returns a copy with bold set to v.
func (s Style) Bold(v bool) Style {
	s.bold = v
	return s
}

// Foreground returns a copy painted with c.
func (s Style) Foreground(c TerminalColor) Style {
	s.fg = c
	return s
}

// Background returns a copy filled with c.
func (s Style) Background(c TerminalColor) Style {
	s.bg = c
	return s
}

// GetForeground is the foreground color, or NoColor when none is set.
func (s Style) GetForeground() TerminalColor { return orNoColor(s.fg) }

// GetBackground is the background color, or NoColor when none is set.
func (s Style) GetBackground() TerminalColor { return orNoColor(s.bg) }

func orNoColor(c TerminalColor) TerminalColor {
	if c == nil {
		return NoColor{}
	}
	return c
}

// Render wraps text in the style's escape sequences.
func (s Style) Render(text string) string {
	codes := s.codes()
	if len(codes) == 0 {
		return text
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}

func (s Style) codes() []string {
	codes := []string{}
	if s.bold {
		codes = append(codes, "1")
	}
	if !colorsEnabled() {
		return codes
	}
	for _, seq := range []string{s.GetForeground().sequence(true), s.GetBackground().sequence(false)} {
		if seq != "" {
			codes = append(codes, seq)
		}
	}
	return codes
}
