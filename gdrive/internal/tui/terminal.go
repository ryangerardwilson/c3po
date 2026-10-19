package tui

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
		return nil, fmt.Errorf("gdrive navigator needs a terminal: %w", err)
	}
	_, _ = os.Stdout.WriteString("\x1b[?1049h\x1b[?25l")
	return &terminal{in: os.Stdin, out: os.Stdout, restore: restore}, nil
}

func (t *terminal) close() {
	_, _ = t.out.WriteString("\x1b[?25h\x1b[?1049l")
	t.restore()
}

func (t *terminal) size() (int, int) {
	return termSize(int(t.out.Fd()))
}

// paint draws one frame from the top left. The last line has no newline
// so a full screen does not scroll; each line is erased to its end.
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
