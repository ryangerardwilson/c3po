package desk

import (
	"errors"
	"fmt"
	"time"

	"github.com/ryangerardwilson/c3po/gcal/internal/event"
)

// Calendar is the primary calendar of one authorized account.
type Calendar interface {
	// ListUpcoming lists up to count single events from now, by start.
	ListUpcoming(count int, now time.Time) ([]event.Event, error)
	Get(id string) (event.Event, error)
	// Delete removes an event and notifies its attendees.
	Delete(id string) error
	// Create adds an event with invitees and a new Meet link.
	Create(title string, start, end time.Time, timezone string, invitees []string, requestID string) (event.Event, error)
	// Reschedule moves an event and notifies its attendees.
	Reschedule(id string, start, end time.Time, timezone string) (event.Event, error)
}

// Drive reads the transcript documents attached to events.
type Drive interface {
	// ExportText returns a document as plain text.
	ExportText(fileID string) (string, error)
}

// Connector opens the Google services of one configured account with a
// usable token, refreshing it or authorizing in the browser when needed.
type Connector interface {
	Calendar(email, clientSecretFile string) (Calendar, error)
	// Drive also grants Meet transcript reads.
	Drive(email, clientSecretFile string) (Drive, error)
}

// Authorizer runs the browser authorization for a client secret, saves
// the token, and returns the authorized account email.
type Authorizer interface {
	Authorize(clientSecretFile string) (email string, err error)
}

// Tokens reports whether an account has a saved token.
type Tokens interface {
	Exists(email string) (bool, error)
}

// Editor opens an editor command line and waits for it to exit. It
// returns ErrEditorNotFound or an *EditorExitError for those failures.
type Editor interface {
	Edit(argv []string) error
}

// EditorFunc is a function that is an Editor.
type EditorFunc func(argv []string) error

// Edit calls f.
func (f EditorFunc) Edit(argv []string) error { return f(argv) }

// ErrEditorNotFound means the editor command does not exist.
var ErrEditorNotFound = errors.New("editor not found")

// EditorExitError is an editor that exited with a nonzero code.
type EditorExitError struct {
	Code int
}

func (e *EditorExitError) Error() string { return fmt.Sprintf("editor exited with code %d", e.Code) }
