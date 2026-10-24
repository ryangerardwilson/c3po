// Package clip puts text on the Wayland clipboard with wl-copy.
package clip

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Clipboard receives copied text. Tests replace it with a fake.
type Clipboard interface {
	Copy(text string, stderr io.Writer) error
}

// WLCopy pipes text to wl-copy. The text is never an argument.
type WLCopy struct{}

// Copy sends text to wl-copy on stdin.
// wl-copy forks and the child serves the clipboard. A stderr Writer that is
// not an *os.File is fed by a pipe, and Wait stays blocked while that child
// holds the pipe, so the screen never reads another key.
func (WLCopy) Copy(text string, stderr io.Writer) error {
	if _, err := exec.LookPath("wl-copy"); err != nil {
		return errors.New("wl-copy is required.")
	}
	cmd := exec.Command("wl-copy")
	cmd.Stdin = strings.NewReader(text)
	if file, ok := stderr.(*os.File); ok {
		cmd.Stderr = file
	}
	if err := cmd.Run(); err != nil {
		return errors.New("Unable to copy to the clipboard.")
	}
	return nil
}
