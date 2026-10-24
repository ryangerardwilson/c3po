// Package slack is the c3po Slack user CLI.
package slack

import (
	"errors"
	"fmt"
	"io"

	"github.com/ryangerardwilson/c3po/slack/internal/app"
)

// Run runs c3po slack with the arguments after "slack".
func Run(args []string, stdout, stderr io.Writer) int {
	rt := app.NewRuntime()
	rt.Stdout = stdout
	rt.Stderr = stderr
	err := rt.Run(args)
	if err == nil {
		return 0
	}
	var usage app.UsageError
	if errors.As(err, &usage) {
		fmt.Fprintln(stderr, usage.Message)
		return 2
	}
	fmt.Fprintln(stderr, err)
	return 1
}
