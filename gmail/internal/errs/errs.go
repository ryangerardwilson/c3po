// Package errs holds the gmail error kinds and their exit codes.
package errs

import (
	"errors"
	"fmt"
)

// Exit codes match the original gmail CLI.
const (
	CodeGeneric = 1
	CodeUsage   = 2
	CodeConfig  = 3
	CodeAPI     = 4
)

// Error is a gmail error with an exit code.
type Error struct {
	Msg  string
	Code int
}

func (e *Error) Error() string { return e.Msg }

// Usage reports bad command usage (exit 2).
func Usage(format string, a ...any) error {
	return &Error{Msg: sprintf(format, a...), Code: CodeUsage}
}

// Config reports an invalid or missing config (exit 3).
func Config(format string, a ...any) error {
	return &Error{Msg: sprintf(format, a...), Code: CodeConfig}
}

// API reports a Gmail or OAuth failure (exit 4).
func API(format string, a ...any) error {
	return &Error{Msg: sprintf(format, a...), Code: CodeAPI}
}

// Systemctl reports a failed systemctl call (exit 2).
type Systemctl struct{ Msg string }

func (e *Systemctl) Error() string { return e.Msg }

// Report maps err to the line printed on stderr and the exit code.
func Report(err error) (string, int) {
	var ge *Error
	if errors.As(err, &ge) {
		return "error: " + ge.Msg, ge.Code
	}
	var se *Systemctl
	if errors.As(err, &se) {
		return "systemctl failed: " + se.Msg, CodeUsage
	}
	return "error: " + err.Error(), CodeGeneric
}

// IsGmail reports whether err is a gmail Error (any code).
func IsGmail(err error) bool {
	var ge *Error
	return errors.As(err, &ge)
}

func sprintf(format string, a ...any) string {
	if len(a) == 0 {
		return format
	}
	return fmt.Sprintf(format, a...)
}
