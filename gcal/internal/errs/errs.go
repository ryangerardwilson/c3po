// Package errs holds the gcal error that maps to a process exit code.
package errs

import "errors"

// Error is a user-facing gcal failure. Its message is printed alone on
// stderr and the process exits with Code.
type Error struct {
	Msg  string
	Code int
}

func (e *Error) Error() string { return e.Msg }

// Usage reports a bad command shape or argument.
func Usage(msg string) error { return &Error{Msg: msg, Code: 2} }

// Config reports a bad or missing config file.
func Config(msg string) error { return &Error{Msg: msg, Code: 2} }

// API reports a failed Google API call.
func API(msg string) error { return &Error{Msg: msg, Code: 2} }

// As returns the gcal error inside err, if any.
func As(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}
