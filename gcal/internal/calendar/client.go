package calendar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/gcal/internal/errs"
)

// Default API roots.
const (
	CalendarBase = "https://www.googleapis.com/calendar/v3"
	DriveBase    = "https://www.googleapis.com/drive/v3"
)

// Client calls one Google API root with a bearer token.
type Client struct {
	HTTP  *http.Client
	Base  string
	Token string
}

// HTTPError is a non-2xx API answer, printed like googleapiclient's.
type HTTPError struct {
	Status  int
	URL     string
	Reason  string
	Details string
}

func (e *HTTPError) Error() string {
	if e.Details == "" {
		return fmt.Sprintf("<HttpError %d when requesting %s returned \"%s\">", e.Status, e.URL, e.Reason)
	}
	return fmt.Sprintf("<HttpError %d when requesting %s returned \"%s\". Details: \"%s\">", e.Status, e.URL, e.Reason, e.Details)
}

// do sends one request and returns the body. A non-2xx answer is an
// *HTTPError; a transport failure is returned as is.
func (c Client) do(method, path string, query url.Values, body any) ([]byte, error) {
	u := strings.TrimRight(c.Base, "/") + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, u, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.send(req)
}

func (c Client) send(req *http.Request) ([]byte, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, httpError(resp, req.URL.String(), data)
	}
	return data, nil
}

func httpError(resp *http.Response, u string, data []byte) *HTTPError {
	e := &HTTPError{Status: resp.StatusCode, URL: u, Reason: http.StatusText(resp.StatusCode)}
	var payload struct {
		Error struct {
			Message string          `json:"message"`
			Errors  json.RawMessage `json:"errors"`
			Details json.RawMessage `json:"details"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &payload) != nil {
		return e
	}
	if payload.Error.Message != "" {
		e.Reason = payload.Error.Message
	}
	switch {
	case len(payload.Error.Details) > 0:
		e.Details = string(payload.Error.Details)
	case len(payload.Error.Errors) > 0:
		e.Details = string(payload.Error.Errors)
	}
	return e
}

// wrap turns an HTTP error into the gcal API error "<what> failed: ...".
// Transport errors pass through unchanged.
func wrap(what string, err error) error {
	if he, ok := err.(*HTTPError); ok {
		return errs.API(fmt.Sprintf("%s failed: %s", what, he.Error()))
	}
	return err
}

func decodeEvent(data []byte) (Event, error) {
	var ev Event
	err := json.Unmarshal(data, &ev)
	return ev, err
}

// ListUpcoming lists up to count single events from now on the primary
// calendar, ordered by start, and keeps those passing mode.
func (c Client) ListUpcoming(count int, mode string, now time.Time) ([]Event, error) {
	q := url.Values{
		"timeMin":      {pyUTC(now)},
		"maxResults":   {fmt.Sprint(count)},
		"singleEvents": {"true"},
		"orderBy":      {"startTime"},
		"alt":          {"json"},
	}
	data, err := c.do(http.MethodGet, "/calendars/primary/events", q, nil)
	if err != nil {
		return nil, wrap("Calendar list", err)
	}
	var payload struct {
		Items []Event `json:"items"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	out := []Event{}
	for _, ev := range payload.Items {
		if ev.Matches(mode) {
			out = append(out, ev)
		}
	}
	return out, nil
}

// pyUTC is Python's datetime.now(UTC).isoformat() with +00:00 as Z.
func pyUTC(t time.Time) string {
	t = t.UTC()
	s := t.Format("2006-01-02T15:04:05")
	if us := t.Nanosecond() / 1000; us != 0 {
		s += fmt.Sprintf(".%06d", us)
	}
	return s + "Z"
}

func eventPath(id string) string {
	return "/calendars/primary/events/" + url.PathEscape(id)
}

// Get fetches one event from the primary calendar.
func (c Client) Get(id string) (Event, error) {
	data, err := c.do(http.MethodGet, eventPath(id), url.Values{"alt": {"json"}}, nil)
	if err != nil {
		return Event{}, wrap("Calendar get", err)
	}
	return decodeEvent(data)
}

// Delete removes one event and notifies attendees.
func (c Client) Delete(id string) error {
	_, err := c.do(http.MethodDelete, eventPath(id), url.Values{"sendUpdates": {"all"}, "alt": {"json"}}, nil)
	return wrap("Calendar delete", err)
}

// TimePayload is a start or end body. timeZone is sent only for IANA
// names, which contain a slash.
func TimePayload(t time.Time, timezone string) map[string]string {
	out := map[string]string{"dateTime": t.Format("2006-01-02T15:04:05-07:00")}
	if strings.Contains(timezone, "/") {
		out["timeZone"] = timezone
	}
	return out
}

// Create inserts an event with invitees and a new Meet link, and
// notifies attendees.
func (c Client) Create(title string, start, end time.Time, timezone string, invitees []string, requestID string) (Event, error) {
	attendees := make([]map[string]string, len(invitees))
	for i, email := range invitees {
		attendees[i] = map[string]string{"email": email}
	}
	body := map[string]any{
		"summary":   title,
		"start":     TimePayload(start, timezone),
		"end":       TimePayload(end, timezone),
		"attendees": attendees,
		"conferenceData": map[string]any{"createRequest": map[string]any{
			"requestId":             requestID,
			"conferenceSolutionKey": map[string]string{"type": "hangoutsMeet"},
		}},
	}
	q := url.Values{"conferenceDataVersion": {"1"}, "sendUpdates": {"all"}, "alt": {"json"}}
	data, err := c.do(http.MethodPost, "/calendars/primary/events", q, body)
	if err != nil {
		return Event{}, wrap("Calendar create", err)
	}
	return decodeEvent(data)
}

// Reschedule moves an event and notifies attendees.
func (c Client) Reschedule(id string, start, end time.Time, timezone string) (Event, error) {
	body := map[string]any{"start": TimePayload(start, timezone), "end": TimePayload(end, timezone)}
	q := url.Values{"sendUpdates": {"all"}, "conferenceDataVersion": {"1"}, "alt": {"json"}}
	data, err := c.do(http.MethodPatch, eventPath(id), q, body)
	if err != nil {
		return Event{}, wrap("Calendar reschedule", err)
	}
	return decodeEvent(data)
}

// ExportText exports a Google Doc from Drive as plain text. Bytes that
// are not UTF-8 become U+FFFD.
func (c Client) ExportText(fileID string) (string, error) {
	q := url.Values{"mimeType": {"text/plain"}}
	data, err := c.do(http.MethodGet, "/files/"+url.PathEscape(fileID)+"/export", q, nil)
	if err != nil {
		return "", wrap("Transcript export", err)
	}
	return strings.ToValidUTF8(string(data), "�"), nil
}
