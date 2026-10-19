package cyclotree

import (
	"os"
	"strings"
)

// screenBytes is one frame. The last line has no newline, so a full
// screen does not scroll. Each line is erased to the end.
func screenBytes(frame string) string {
	lines := strings.Split(frame, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	var b strings.Builder
	b.WriteString("\x1b[?2026h\x1b[H\x1b[38;2;220;220;0m")
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
		b.WriteString("\x1b[K")
	}
	b.WriteString("\x1b[J\x1b[0m\x1b[?2026l")
	return b.String()
}

func paint(out *os.File, frame string) {
	if out == nil {
		return
	}
	_, _ = out.WriteString(screenBytes(frame))
}

// decodeSeq maps a key or an arrow sequence to a motion.
// A zero result is ignored. rest is the bytes after an escape.
func decodeSeq(first byte, rest string) rune {
	if first != 0x1b {
		return rune(first)
	}
	switch rest {
	case "[A", "OA":
		return 'k'
	case "[B", "OB":
		return 'j'
	case "[C", "OC":
		return 'l'
	case "[D", "OD":
		return 'h'
	default:
		return 0
	}
}
