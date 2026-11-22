// Package slack is the c3po Slack user CLI.
package slack

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ryangerardwilson/c3po/slack/internal/app"
	"github.com/ryangerardwilson/c3po/slack/internal/cachedb"
	"github.com/ryangerardwilson/c3po/slack/internal/slackapi"
	"github.com/ryangerardwilson/c3po/slack/internal/tty"
)

// Run runs c3po slack with the arguments after "slack".
func Run(args []string, stdout, stderr io.Writer) int {
	rt := newRuntime(stdout, stderr, &http.Client{Timeout: 60 * time.Second}, slackapi.DefaultBaseURL)
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

// newRuntime composes the core with the Slack Web API at baseURL over
// httpClient, the sqlite event cache, and the process terminal check.
func newRuntime(stdout, stderr io.Writer, httpClient *http.Client, baseURL string) *app.Runtime {
	rt := app.NewRuntime()
	rt.Stdout = stdout
	rt.Stderr = stderr
	rt.Slack = slackapi.Connector{HTTPClient: httpClient, BaseURL: baseURL}
	rt.Cache = cachedb.Store{}
	rt.IsTTY = tty.IsInteractive
	rt.StdioIsTerminal = tty.IsInteractive
	rt.RunEditor = tty.RunEditor
	return rt
}
