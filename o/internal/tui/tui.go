// Package tui is a small Elm-style terminal runtime: a model receives
// messages, returns commands, and renders a view inline in the terminal.
package tui

import (
	"errors"
	"os/exec"
	"time"
)

// Msg is any event delivered to Model.Update.
type Msg interface{}

// Cmd runs off the event loop and returns one message, or nil.
type Cmd func() Msg

// Model is the state of one screen.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() string
}

// WindowSizeMsg reports the terminal size at start and after a resize.
type WindowSizeMsg struct {
	Width  int
	Height int
}

// QuitMsg ends the program after the final frame.
type QuitMsg struct{}

// Quit is a command that ends the program.
func Quit() Msg { return QuitMsg{} }

// ErrInterrupted is returned by Program.Run after SIGINT.
var ErrInterrupted = errors.New("program was interrupted")

type interruptMsg struct{}

// BatchMsg runs several commands at once.
type BatchMsg []Cmd

// Batch combines commands. Nil commands are dropped.
func Batch(cmds ...Cmd) Cmd {
	valid := make([]Cmd, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd != nil {
			valid = append(valid, cmd)
		}
	}
	switch len(valid) {
	case 0:
		return nil
	case 1:
		return valid[0]
	}
	return func() Msg { return BatchMsg(valid) }
}

// Tick waits d, then returns fn of the time it fired.
func Tick(d time.Duration, fn func(time.Time) Msg) Cmd {
	return func() Msg {
		timer := time.NewTimer(d)
		defer timer.Stop()
		return fn(<-timer.C)
	}
}

type execMsg struct {
	cmd *exec.Cmd
	fn  func(error) Msg
}

// Exec hands the terminal to cmd, waits for it, and then returns fn of
// its error. Unset stdin, stdout, and stderr are attached to the terminal.
func Exec(cmd *exec.Cmd, fn func(error) Msg) Cmd {
	return func() Msg { return execMsg{cmd: cmd, fn: fn} }
}
