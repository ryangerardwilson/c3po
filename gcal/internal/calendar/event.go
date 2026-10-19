// Package calendar calls the Google Calendar v3 and Drive v3 REST APIs.
package calendar

import "strings"

// Event is the subset of a Calendar event gcal reads.
type Event struct {
	ID               string       `json:"id"`
	Summary          *string      `json:"summary"`
	Start            EventTime    `json:"start"`
	End              EventTime    `json:"end"`
	Attendees        []Attendee   `json:"attendees"`
	HangoutLink      string       `json:"hangoutLink"`
	ConferenceData   *Conference  `json:"conferenceData"`
	RecurringEventID string       `json:"recurringEventId"`
	Recurrence       []string     `json:"recurrence"`
	Attachments      []Attachment `json:"attachments"`
}

// EventTime is a timed or all-day boundary.
type EventTime struct {
	DateTime string `json:"dateTime"`
	Date     string `json:"date"`
}

// Value is dateTime, or date for an all-day event.
func (t EventTime) Value() string {
	if t.DateTime != "" {
		return t.DateTime
	}
	return t.Date
}

// Attendee is one invitee.
type Attendee struct {
	Email string `json:"email"`
}

// Conference holds the Meet entry points.
type Conference struct {
	EntryPoints []EntryPoint `json:"entryPoints"`
}

// EntryPoint is one way to join a conference.
type EntryPoint struct {
	EntryPointType string `json:"entryPointType"`
	URI            string `json:"uri"`
}

// Attachment is a Drive file attached to the event.
type Attachment struct {
	FileID   string `json:"fileId"`
	Title    string `json:"title"`
	MimeType string `json:"mimeType"`
	FileURL  string `json:"fileUrl"`
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
