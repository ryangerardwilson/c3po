package gmail

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const barCells = 16

// cleanBar is the cleanup progress line. A terminal redraws it in place.
// status is the preset text at the end of the line.
type cleanBar struct {
	out    io.Writer
	label  string
	status string
	total  int
	live   bool
	color  bool
}

func newCleanBar(out io.Writer, label string, total int) cleanBar {
	live := writerTerminal(out)
	return cleanBar{
		out:   out,
		label: label,
		total: total,
		live:  live,
		color: barColor(live),
	}
}

func writerTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func barColor(live bool) bool {
	return live && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
}

func (b cleanBar) start() { b.draw(0, false) }

func (b cleanBar) step(done int) { b.draw(done, false) }

func (b cleanBar) finish(trashed int) {
	b.status = fmt.Sprintf("trashed %d", trashed)
	b.draw(b.total, true)
}

func (b cleanBar) hold() { b.draw(b.total, true) }

func (b cleanBar) fail() {
	if b.live {
		fmt.Fprintln(b.out)
	}
}

func (b cleanBar) draw(done int, hold bool) {
	if !b.live && !hold {
		return
	}
	filled := 0
	if b.total > 0 && done > 0 {
		filled = barCells * done / b.total
		if filled == 0 {
			filled = 1
		}
		if done >= b.total || filled > barCells {
			filled = barCells
		}
	}
	on := strings.Repeat("█", filled)
	off := strings.Repeat("░", barCells-filled)
	blocks := on + off
	if b.color {
		blocks = "\x1b[38;5;42m" + on + "\x1b[38;5;238m" + off + "\x1b[0m"
	}
	line := fmt.Sprintf("%s  %d/%d", blocks, done, b.total)
	if b.label != "" {
		line = b.label + "  " + line
	}
	if b.status != "" {
		line += "  " + b.status
	}
	if b.live {
		fmt.Fprintf(b.out, "\r\x1b[K%s", line)
		if hold {
			fmt.Fprintln(b.out)
		}
		return
	}
	fmt.Fprintln(b.out, line)
}
