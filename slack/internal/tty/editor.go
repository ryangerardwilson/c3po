package tty

import (
	"os"
	"os/exec"
)

// RunEditor runs editor on path through sh. It attaches only to the real
// process terminals, never to captured output that could leak the
// token-bearing file into agent logs.
func RunEditor(editor, path string) error {
	cmd := exec.Command("sh", "-c", editor+" \"$1\"", "slack-editor", path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
