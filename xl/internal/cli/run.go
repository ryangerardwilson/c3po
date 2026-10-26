package cli

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"

	"github.com/ryangerardwilson/c3po/internal/tuisession"
	"github.com/ryangerardwilson/c3po/xl/internal/sheet"
	"github.com/ryangerardwilson/c3po/xl/internal/tui"
	"github.com/ryangerardwilson/c3po/xl/internal/version"
)

type Runner struct {
	Out      io.Writer
	Err      io.Writer
	Terminal func() bool
	resume   string
}

func NewRunner(out, errOut io.Writer) Runner {
	return Runner{Out: out, Err: errOut}
}

func (r Runner) Run(args []string) int {
	id, rest, err := tuisession.TakeResume(args)
	if err != nil {
		return r.fail(err.Error())
	}
	r.resume = id
	args = rest
	if len(args) == 0 {
		if !r.terminal() {
			WriteHelp(r.Out)
			return 0
		}
		return r.open("")
	}
	switch args[0] {
	case "help":
		if len(args) != 1 {
			return r.fail("help takes no arguments")
		}
		WriteHelp(r.Out)
		return 0
	case "version":
		if len(args) != 1 {
			return r.fail("version takes no arguments")
		}
		_, _ = fmt.Fprintln(r.Out, version.Version)
		return 0
	case "open":
		switch len(args) {
		case 1:
			return r.open("")
		case 2:
			return r.open(args[1])
		default:
			return r.fail("open takes a path or nothing")
		}
	default:
		if len(args) != 1 {
			return r.fail("xl takes one file name\n\nRun: c3po tui xl help")
		}
		return r.open(args[0])
	}
}

func (r Runner) open(path string) int {
	if !r.terminal() {
		return r.fail("open needs a terminal")
	}
	sess, err := tuisession.Open("xl", r.resume)
	if err != nil {
		return r.fail(err.Error())
	}
	defer sess.Close()
	if path == "" {
		if body, ok := sess.Restored(); ok && body.Path != "" {
			path = body.Path
		}
	}
	if err := sess.Note(tuisession.Body{Path: path}); err != nil {
		return r.fail(err.Error())
	}
	s, err := sheet.Open(path)
	if err != nil {
		return r.fail(err.Error())
	}
	if err := tui.Run(s); err != nil {
		return r.fail(err.Error())
	}
	return 0
}

func (r Runner) terminal() bool {
	if r.Terminal != nil {
		return r.Terminal()
	}
	file, ok := r.Out.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func (r Runner) fail(msg string) int {
	fmt.Fprintf(r.Err, "xl: %s\n", msg)
	return 1
}
