package deport

import (
	"fmt"
	"os"
	"strings"
)

type terminal struct {
	in      *os.File
	out     *os.File
	restore func()
}

func openTerminal() (*terminal, error) {
	restore, err := enterRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("deport screen needs a terminal: %w", err)
	}
	_, _ = os.Stdout.WriteString("\x1b[?1049h\x1b[?25l")
	return &terminal{in: os.Stdin, out: os.Stdout, restore: restore}, nil
}

func (t *terminal) close() {
	_, _ = t.out.WriteString("\x1b[?25h\x1b[?1049l")
	if t.restore != nil {
		t.restore()
		t.restore = nil
	}
}

func (t *terminal) size() (int, int) {
	return termSize(int(t.out.Fd()))
}

func (t *terminal) paint(frame string) {
	lines := strings.Split(strings.TrimSuffix(frame, "\n"), "\n")
	var b strings.Builder
	b.WriteString("\x1b[H")
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
		b.WriteString("\x1b[K")
	}
	b.WriteString("\x1b[J")
	_, _ = t.out.WriteString(b.String())
}

func (t *terminal) readKey() (string, error) {
	buf := make([]byte, 16)
	n, err := t.in.Read(buf)
	if err != nil {
		return "", err
	}
	return decodeKey(buf[:n]), nil
}

var escapeKeys = map[string]string{
	"[A": "up", "OA": "up",
	"[B": "down", "OB": "down",
	"[C": "right", "OC": "right",
	"[D": "left", "OD": "left",
	"[H": "home", "OH": "home", "[1~": "home", "[7~": "home",
	"[F": "end", "OF": "end", "[4~": "end", "[8~": "end",
}

func decodeKey(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if data[0] == 0x1b && len(data) > 1 {
		return escapeKeys[string(data[1:])]
	}
	switch data[0] {
	case 0x03:
		return "ctrl+c"
	case 0x1b:
		return "esc"
	case '\r', '\n':
		return "enter"
	case 0x7f, 0x08:
		return "backspace"
	default:
		return string(data[:1])
	}
}
