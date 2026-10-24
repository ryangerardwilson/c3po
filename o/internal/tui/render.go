package tui

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// renderer paints frames inline, below the shell prompt, rewriting the
// lines of the previous frame in place.
type renderer struct {
	out    io.Writer
	width  int
	height int
	lines  int
	last   string
}

func (r *renderer) resize(width, height int) {
	r.width = width
	r.height = height
	r.repaint()
}

// repaint forces the next frame to be written even when unchanged.
func (r *renderer) repaint() {
	r.last = ""
}

func (r *renderer) paint(view string) {
	if view == r.last && r.last != "" {
		return
	}
	_, _ = io.WriteString(r.out, r.frame(view))
	r.last = view
}

func (r *renderer) frame(view string) string {
	lines := strings.Split(view, "\n")
	if r.height > 0 && len(lines) > r.height {
		lines = lines[len(lines)-r.height:]
	}
	for r.height > 0 && len(lines) < r.height {
		lines = append(lines, "")
	}
	var b strings.Builder
	if r.lines > 1 {
		fmt.Fprintf(&b, "\x1b[%dA", r.lines-1)
	}
	b.WriteByte('\r')
	for i, line := range lines {
		if i > 0 {
			b.WriteString("\r\n")
		}
		b.WriteString(fitLine(line, r.width))
	}
	if r.lines > len(lines) {
		b.WriteString("\x1b[J")
	}
	b.WriteByte('\r')
	r.lines = len(lines)
	return b.String()
}

// stop leaves the final frame on screen without its last line.
func (r *renderer) stop() {
	_, _ = io.WriteString(r.out, "\x1b[2K\r")
}

// fitLine cuts a line to width visible cells, keeping escape sequences,
// and erases the rest of the row when the line is shorter.
func fitLine(line string, width int) string {
	if width <= 0 {
		return line
	}
	var b strings.Builder
	cells := 0
	for i := 0; i < len(line); {
		if n := escapeLen(line[i:]); n > 0 {
			b.WriteString(line[i : i+n])
			i += n
			continue
		}
		_, size := utf8.DecodeRuneInString(line[i:])
		if cells < width {
			b.WriteString(line[i : i+size])
		}
		cells++
		i += size
	}
	if cells < width {
		b.WriteString("\x1b[K")
	}
	return b.String()
}

// escapeLen is the length of a CSI sequence at the start of s, or 0.
func escapeLen(s string) int {
	if len(s) < 2 || s[0] != 0x1b || s[1] != '[' {
		return 0
	}
	for i := 2; i < len(s); i++ {
		if s[i] >= 0x40 && s[i] <= 0x7e {
			return i + 1
		}
	}
	return len(s)
}
