package gmail

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const barCells = 16

// cleanBar is one preset's cleanup line. A terminal redraws it in place.
type cleanBar struct {
	out   io.Writer
	label string
	total int
	live  bool
	color bool
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

func (b cleanBar) start() { b.draw(0, -1, false) }

func (b cleanBar) step(done int) { b.draw(done, -1, false) }

func (b cleanBar) finish(trashed int) { b.draw(b.total, trashed, true) }

func (b cleanBar) fail() {
	if b.live {
		fmt.Fprintln(b.out)
	}
}

func (b cleanBar) draw(done, trashed int, hold bool) {
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
	bar := on + off
	if b.color {
		bar = "\x1b[38;5;42m" + on + "\x1b[38;5;238m" + off + "\x1b[0m"
	}
	line := fmt.Sprintf("%s  %s  %d/%d", b.label, bar, done, b.total)
	if trashed >= 0 {
		line += fmt.Sprintf("  trashed %d", trashed)
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
