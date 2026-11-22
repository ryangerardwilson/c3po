// Package desk holds the gcal commands and their rules: dispatch,
// presets, list, create, preview, delete, reschedule, transcript export,
// accounts, setup check, auth, and config. It reaches Google, the token
// store, and the editor only through the ports in ports.go, which the
// gcal package wires.
package desk

import (
	"bufio"
	"fmt"
	"io"
	"time"

	"github.com/ryangerardwilson/c3po/gcal/internal/errs"
	"github.com/ryangerardwilson/c3po/gcal/internal/paths"
	"github.com/ryangerardwilson/c3po/internal/version"
)

const (
	ansiGray  = "\x1b[38;5;245m"
	ansiReset = "\x1b[0m"
)

// Env is everything a command touches outside its arguments.
type Env struct {
	Lookup     func(string) (string, bool)
	Stdin      *bufio.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	Now        func() time.Time
	NewID      func() string
	TTY        bool
	Editor     Editor
	Google     Connector
	Authorizer Authorizer
	Tokens     Tokens
}

func (e *Env) getenv(key string) string {
	v, _ := e.Lookup(key)
	return v
}

func (e *Env) dirs() paths.Dirs { return paths.Resolve(e.getenv) }

// Run runs one gcal command and returns its exit code. A gcal error is
// printed alone on stderr; any other error is printed after "gcal: ".
func Run(args []string, e *Env) int {
	err := dispatchTop(args, e)
	if err == nil {
		return 0
	}
	if ge, ok := errs.As(err); ok {
		fmt.Fprintln(e.Stderr, ge.Msg)
		return ge.Code
	}
	fmt.Fprintf(e.Stderr, "gcal: %s\n", err.Error())
	return 1
}

func dispatchTop(args []string, e *Env) error {
	if len(args) == 0 || (len(args) == 1 && isHelp(args[0])) {
		printHelp(e)
		return nil
	}
	if len(args) == 1 && args[0] == "version" {
		fmt.Fprintln(e.Stdout, version.String())
		return nil
	}
	if args[0] == "upgrade" {
		return errs.Usage("upgrade is not part of c3po gcal; upgrade c3po itself")
	}
	if isHelp(args[0]) || args[0] == "version" {
		return errs.Usage("Use: c3po gcal " + args[0])
	}
	if len(args) == 1 && args[0] == "config" {
		return openConfig(e)
	}
	return dispatch(args, e)
}

func isHelp(s string) bool { return s == "help" || s == "-h" || s == "--help" }

func printHelp(e *Env) {
	_, noColor := e.Lookup("NO_COLOR")
	if !e.TTY || noColor {
		fmt.Fprint(e.Stdout, Help)
		return
	}
	fmt.Fprint(e.Stdout, ansiGray+Help[:len(Help)-1]+ansiReset+"\n")
}
