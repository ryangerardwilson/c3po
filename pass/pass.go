// Package pass is c3po pass, the interface for 1Password items.
// It runs the 1Password CLI, op.
package pass

import (
	"fmt"
	"io"
	"os"

	"github.com/ryangerardwilson/c3po/internal/version"
	"github.com/ryangerardwilson/c3po/pass/internal/cli"
	"github.com/ryangerardwilson/c3po/pass/internal/clip"
	"github.com/ryangerardwilson/c3po/pass/internal/op"
	"github.com/ryangerardwilson/c3po/pass/internal/tui"
)

// Help is the full help for c3po pass.
const Help = `c3po pass copies and changes 1Password items

global actions:
  c3po pass help
    show this help
  c3po pass version
    print the c3po version

  copy a password by title or Login username
  # c3po pass <vault> <search>
  c3po pass Root github

  list items in a vault
  # c3po pass <vault> -l
  c3po pass wiom -l

  copy several secrets as JSON
  # c3po pass <vault> <item> <item>
  c3po pass wiom "WIOM_SNOWFLAKE_*"

  create, edit, show, or delete one exact title
  # c3po pass <vault> --create --title <title> --category api-credential|login
  # c3po pass <vault> --edit <title>
  # c3po pass <vault> --show <title>
  # c3po pass <vault> --delete <title>
  # c3po pass <vault> --create --template <file>
  c3po pass Root --create --title GH_EXAMPLE --category api-credential
  c3po pass Root --edit GH_EXAMPLE
  c3po pass Root --show GH_EXAMPLE
  c3po pass Root --delete GH_EXAMPLE

The item screen is c3po tui pass.
One match is copied with wl-copy and is not printed. -l prints titles and
usernames, never passwords. A secret is read from stdin. The token is
OP_SERVICE_ACCOUNT_TOKEN, or the file named by
OP_SERVICE_ACCOUNT_TOKEN_FILE, or ~/.config/c3po/pass/service-account-token.
`

// Run runs c3po pass. args follow "c3po pass". A secret is
// read from os.Stdin.
func Run(args []string, stdout, stderr io.Writer) int {
	runner := cli.Runner{
		Out:     stdout,
		Err:     stderr,
		In:      os.Stdin,
		Op:      op.Exec{},
		Clip:    clip.WLCopy{},
		Help:    Help,
		Version: version.String(),
	}
	return runner.Run(args)
}

// ScreenHelp is the help for c3po tui pass.
const ScreenHelp = `c3po tui pass
browse 1Password vaults and copy one item

  c3po tui pass
    list vaults, open one, and copy an item
  c3po tui pass help
    show this help

j and k move. l opens a vault. h goes back.
enter copies the selected item. The secret is not printed.
dd deletes the selected item.
q quits.
create, edit, show, and search stay on c3po pass.
`

// Screen lists vaults, then copies one item. The secret is not printed.
func Screen(stdout, stderr io.Writer) int {
	env, err := op.Env()
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	runner := op.Exec{}
	if err := runner.LookPath(); err != nil {
		fmt.Fprintln(stderr, "op is required.")
		return 1
	}
	client := op.Client{Runner: runner, Env: env, Err: stderr}
	if err := tui.Run(client, clip.WLCopy{}); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}
