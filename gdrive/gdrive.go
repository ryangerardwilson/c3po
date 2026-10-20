// Package gdrive is the c3po for local-first Google Drive sync.
package gdrive

import (
	"fmt"
	"io"

	"github.com/ryangerardwilson/c3po/gdrive/internal/cli"
	"github.com/ryangerardwilson/c3po/internal/version"
)

// Help is the full help for c3po gdrive.
const Help = `c3po gdrive syncs local folders with Google Drive.

Usage:
  c3po gdrive help
  c3po gdrive version
  c3po gdrive auth <client_secret_path>
  c3po gdrive config
  c3po gdrive <preset> register <local_dir> as <drive_path>
  c3po gdrive <preset> list registrations
  c3po gdrive <preset> remove registration <id>
  c3po gdrive <preset> restore registration <id>
  c3po gdrive <preset> restore registrations
  c3po gdrive sync run
  c3po gdrive sync restore
  c3po gdrive timer install|disable|status

auth saves a Google account as the next numbered preset. register remembers
a local directory and the Drive folder it syncs with. sync run is what the
hourly timer runs. sync restore pulls registered folders back to disk.
The Drive screen is c3po tui gdrive.

  c3po gdrive auth ~/Documents/credentials/client_secret.json
  c3po gdrive 1 register ~/Documents as Documents
  c3po gdrive sync run

help, -h, and --help print this text. version, -v, and --version print the
c3po version. A usage error exits 2.
`

// ScreenHelp is the help for c3po tui gdrive.
const ScreenHelp = `c3po tui gdrive
browse or upload one Google Drive preset

  c3po tui gdrive
    choose a preset, then browse or upload
  c3po tui gdrive help
    show this help

j and k move. enter selects. h goes back from the action list.
browse opens that preset's Drive. upload asks for local paths,
then opens Drive to pick the destination. q quits.
auth, sync, and registration stay on c3po gdrive.
`

// Screen asks for a preset and an action, then opens that Drive screen.
func Screen(stdout, stderr io.Writer) int {
	return cli.Screen(stdout, stderr)
}

// Run runs c3po gdrive with the arguments after "gdrive".
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args) {
		fmt.Fprint(stdout, Help)
		return 0
	}
	if isVersion(args) {
		fmt.Fprintln(stdout, "c3po "+version.String())
		return 0
	}
	if err := cli.Dispatch(args, stdout, stderr); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	return 0
}

func isHelp(args []string) bool {
	return len(args) == 1 && (args[0] == "help" || args[0] == "-h" || args[0] == "--help")
}

func isVersion(args []string) bool {
	return len(args) == 1 && (args[0] == "version" || args[0] == "-v" || args[0] == "--version")
}
