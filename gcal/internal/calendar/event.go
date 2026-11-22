// Package calendar calls the Google Calendar v3 and Drive v3 REST APIs.
package calendar

import "github.com/ryangerardwilson/c3po/gcal/internal/event"

// wireEvent is the JSON shape of the Calendar event fields gcal reads.
type wireEvent struct {
	ID               string           `json:"id"`
	Summary          *string          `json:"summary"`
	Start            wireTime         `json:"start"`
	End              wireTime         `json:"end"`
	Attendees        []wireAttendee   `json:"attendees"`
	HangoutLink      string           `json:"hangoutLink"`
	ConferenceData   *wireConference  `json:"conferenceData"`
	RecurringEventID string           `json:"recurringEventId"`
	Recurrence       []string         `json:"recurrence"`
	Attachments      []wireAttachment `json:"attachments"`
}

type wireTime struct {
	DateTime string `json:"dateTime"`
	Date     string `json:"date"`
}

type wireAttendee struct {
	Email string `json:"email"`
}

type wireConference struct {
	EntryPoints []wireEntryPoint `json:"entryPoints"`
}

type wireEntryPoint struct {
	EntryPointType string `json:"entryPointType"`
	URI            string `json:"uri"`
}

type wireAttachment struct {
	FileID   string `json:"fileId"`
	Title    string `json:"title"`
	MimeType string `json:"mimeType"`
	FileURL  string `json:"fileUrl"`
}

// domain translates the API event into the gcal event.
func (w wireEvent) domain() event.Event {
	ev := event.Event{
		ID:               w.ID,
		Summary:          w.Summary,
		Start:            event.Time(w.Start),
		End:              event.Time(w.End),
		HangoutLink:      w.HangoutLink,
		RecurringEventID: w.RecurringEventID,
		Recurrence:       w.Recurrence,
	}
	for _, a := range w.Attendees {
		ev.Attendees = append(ev.Attendees, event.Attendee(a))
	}
	if w.ConferenceData != nil {
		conf := &event.Conference{}
		for _, ep := range w.ConferenceData.EntryPoints {
			conf.EntryPoints = append(conf.EntryPoints, event.EntryPoint(ep))
		}
		ev.ConferenceData = conf
	}
	for _, a := range w.Attachments {
		ev.Attachments = append(ev.Attachments, event.Attachment(a))
	}
	return ev
}
