// Package event holds the calendar event gcal reads, in domain terms.
// It imports no transport: the calendar adapter translates its API
// answers into these types.
package event

import "strings"

// Event is the subset of a calendar event gcal reads.
type Event struct {
	ID               string
	Summary          *string
	Start            Time
	End              Time
	Attendees        []Attendee
	HangoutLink      string
	ConferenceData   *Conference
	RecurringEventID string
	Recurrence       []string
	Attachments      []Attachment
}

// Time is a timed or all-day boundary.
type Time struct {
	DateTime string
	Date     string
}

// Value is the date and time, or the date for an all-day event.
func (t Time) Value() string {
	if t.DateTime != "" {
		return t.DateTime
	}
	return t.Date
}

// Attendee is one invitee.
type Attendee struct {
	Email string
}

// Conference holds the Meet entry points.
type Conference struct {
	EntryPoints []EntryPoint
}

// EntryPoint is one way to join a conference.
type EntryPoint struct {
	EntryPointType string
	URI            string
}

// Attachment is a Drive file attached to the event.
type Attachment struct {
	FileID   string
	Title    string
	MimeType string
	FileURL  string
}

// Title is the summary, or fallback when the event has none.
func (e Event) Title(fallback string) string {
	if e.Summary == nil {
		return fallback
	}
	return *e.Summary
}

// MeetingURL is the hangout link, else the first video entry point,
// else "-".
func (e Event) MeetingURL() string {
	if link := strings.TrimSpace(e.HangoutLink); link != "" {
		return link
	}
	if e.ConferenceData != nil {
		for _, ep := range e.ConferenceData.EntryPoints {
			if strings.TrimSpace(ep.EntryPointType) == "video" {
				return strings.TrimSpace(ep.URI)
			}
		}
	}
	return "-"
}

// AttendeeEmails lists the non-empty attendee emails.
func (e Event) AttendeeEmails() []string {
	out := []string{}
	for _, a := range e.Attendees {
		if email := strings.TrimSpace(a.Email); email != "" {
			out = append(out, email)
		}
	}
	return out
}

// Recurring reports an instance of a series or a series itself.
func (e Event) Recurring() bool {
	return e.RecurringEventID != "" || len(e.Recurrence) > 0
}

// Recurrence filters for listing.
const (
	All          = "all"
	RecurringOn  = "recurring"
	NonRecurring = "non_recurring"
)

// Matches reports whether e passes the recurrence filter mode.
func (e Event) Matches(mode string) bool {
	switch mode {
	case All:
		return true
	case RecurringOn:
		return e.Recurring()
	default:
		return !e.Recurring()
	}
}

// Keep returns the events that pass the recurrence filter mode, never nil.
func Keep(events []Event, mode string) []Event {
	out := []Event{}
	for _, ev := range events {
		if ev.Matches(mode) {
			out = append(out, ev)
		}
	}
	return out
}
