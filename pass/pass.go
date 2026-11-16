// Package pass is the c3po entry for nebula crypt.
// c3po pass runs that feature. r2d2 pass is the item screen and
// reaches 1Password through nebula crypt.
package pass

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/ryangerardwilson/c3po/internal/tuisession"
	"github.com/ryangerardwilson/c3po/pass/internal/tui"
)

// ScreenHelp is the help for r2d2 pass.
const ScreenHelp = `r2d2 pass
browse 1Password vaults and copy one item

  r2d2 pass
    list vaults, open one, and copy an item
  r2d2 pass help
    show this help

j and k move. l opens a vault. h goes back.
enter copies the selected item. The secret is not printed.
dd deletes the selected item.
q quits.
Vaults, items, copy, and delete go through nebula crypt.
create, edit, show, and search stay on nebula crypt.
`

// Run runs nebula crypt. args are the words after pass.
func Run(args []string, stdout, stderr io.Writer) int {
	return execNebula(append([]string{"crypt"}, args...), stdout, stderr)
}

// Screen lists vaults, then copies one item. The secret is not printed.
func Screen(stdout, stderr io.Writer) int {
	return screenWith("", stdout, stderr)
}

// ScreenResume opens the vault list on a saved session. The secret is not stored.
func ScreenResume(id string, stdout, stderr io.Writer) int {
	return screenWith(id, stdout, stderr)
}

func screenWith(id string, stdout, stderr io.Writer) int {
	if _, err := lookNebula("nebula"); err != nil {
		fmt.Fprintln(stderr, "nebula is not installed")
		return 1
	}
	sess, err := tuisession.Open("pass", id)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	defer sess.Close()
	if err := tui.RunSession(tui.Nebula{}, sess); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

var lookNebula = exec.LookPath

var runCommand = func(path string, args []string, stdout, stderr io.Writer) int {
	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	fmt.Fprintln(stderr, err.Error())
	return 1
}

func execNebula(args []string, stdout, stderr io.Writer) int {
	path, err := lookNebula("nebula")
	if err != nil {
		fmt.Fprintln(stderr, "nebula is not installed")
		return 1
	}
	return runCommand(path, args, stdout, stderr)
}
