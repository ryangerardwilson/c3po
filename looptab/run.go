// Package looptab is the c3po tui looptab screen.
package looptab

import (
	"errors"
	"fmt"
	"io"
	"os/exec"

	"github.com/ryangerardwilson/c3po/internal/version"
)

var errNotOnPath = errors.New("looptab is not on PATH")

// ScreenHelp is the help for c3po tui looptab.
const ScreenHelp = `c3po tui looptab
list Looptab jobs and run the selected job now

  c3po tui looptab
    list the jobs from looptab now
  c3po tui looptab help
    show this help

j and k move. enter runs the selected job now.
q quits. ? lists the shortcuts.
The job runs through looptab now and that row's index.
`

// Run serves c3po tui looptab. args are the arguments after the screen name.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && isHelp(args[0]) {
		fmt.Fprint(stdout, ScreenHelp)
		return 0
	}
	if len(args) == 1 && isVersion(args[0]) {
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	}
	if len(args) != 0 {
		fmt.Fprint(stderr, ScreenHelp)
		return 2
	}
	return openScreen(stdout, stderr)
}

func openScreen(_ io.Writer, stderr io.Writer) int {
	bin, lookErr := exec.LookPath("looptab")
	term, err := openTerminal()
	if err != nil {
		fmt.Fprintf(stderr, "looptab: %s\n", err.Error())
		return 1
	}
	defer term.close()
	m := &model{
		load: func() ([]job, error) {
			if lookErr != nil {
				return nil, errNotOnPath
			}
			return loadJobs(bin)
		},
		run: func(j job) error {
			term.pause()
			err := runNow(bin, j.Index)
			if rerr := term.resume(); err == nil {
				err = rerr
			}
			return err
		},
	}
	if err := m.loop(term); err != nil {
		fmt.Fprintf(stderr, "looptab: %s\n", err.Error())
		return 1
	}
	return 0
}

func isHelp(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}

func isVersion(arg string) bool {
	return arg == "version" || arg == "-v" || arg == "--version"
}
