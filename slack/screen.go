package slack

import (
	"fmt"
	"io"

	"github.com/ryangerardwilson/c3po/internal/tuisession"
	"github.com/ryangerardwilson/c3po/internal/version"
	"github.com/ryangerardwilson/c3po/slack/internal/tui"
)

// ScreenHelp is the help for r2d2 slack.
const ScreenHelp = `r2d2 slack
Slack desk. Direct messages, tags, and what to know.

  r2d2 slack
    open the desk
  r2d2 slack help
    show this help

j and k move. h and l switch workspace.
enter opens the row. Opening an item does not mark it read.
r marks the row read on the desk.
q quits. ? lists the shortcuts.
The rows come from the Slack desk. The token is not shown.
`

// Run serves r2d2 slack. args are the arguments after the screen name.
func Screen(args []string, stdout, stderr io.Writer) int {
	id, rest, err := tuisession.TakeResume(args)
	if err != nil {
		fmt.Fprint(stderr, ScreenHelp)
		return 2
	}
	args = rest
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
	if err := tui.Run(id); err != nil {
		fmt.Fprintf(stderr, "slack: %s\n", err.Error())
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
