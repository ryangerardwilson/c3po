// Package tunnel is the c3po entry for nebula tunnel.
// c3po tunnel and r2d2 tunnel run that feature.
package tunnel

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// ScreenHelp is the help for r2d2 tunnel.
const ScreenHelp = `r2d2 tunnel
edit routes in the terminal

  r2d2 tunnel
    open the route screen
  r2d2 tunnel help
    show this help

The route screen is nebula tunnel screen.
j and k move. a adds a route. d asks before removing the selected
route. y confirms that removal and n cancels it. q quits.
`

// Run runs nebula tunnel. args are the words after tunnel.
func Run(args []string, stdout, stderr io.Writer) int {
	return execNebula(append([]string{"tunnel"}, args...), stdout, stderr)
}

// Screen runs the route editor. The command is r2d2 tunnel.
func Screen(stdout, stderr io.Writer) int {
	return ScreenResume("", stdout, stderr)
}

// ScreenResume opens the route editor on a saved session.
func ScreenResume(id string, stdout, stderr io.Writer) int {
	args := []string{"tunnel", "screen"}
	if id != "" {
		args = append(args, "--resume", id)
	}
	return execNebula(args, stdout, stderr)
}

func execNebula(args []string, stdout, stderr io.Writer) int {
	path, err := exec.LookPath("nebula")
	if err != nil {
		fmt.Fprintln(stderr, "nebula is not installed")
		return 1
	}
	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err = cmd.Run(); err == nil {
		return 0
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	fmt.Fprintln(stderr, err.Error())
	return 1
}
