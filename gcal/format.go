package gcal

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/gcal/internal/calendar"
	"github.com/ryangerardwilson/c3po/gcal/internal/config"
	"github.com/ryangerardwilson/c3po/gcal/internal/errs"
)

var timePattern = regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2}) (\d{1,2}):(\d{1,2}):(\d{1,2})$`)

// parseTime reads YYYY-MM-DD HH:MM:SS as wall time in timezone.
func parseTime(value, timezone string) (time.Time, error) {
	bad := errs.Usage("time must use YYYY-MM-DD HH:MM:SS")
	m := timePattern.FindStringSubmatch(value)
	if m == nil {
		return time.Time{}, bad
	}
	n := make([]int, 6)
	for i := range n {
		n[i], _ = strconv.Atoi(m[i+1])
	}
	loc, err := config.Location(timezone)
	if err != nil {
		return time.Time{}, err
	}
	t := time.Date(n[0], time.Month(n[1]), n[2], n[3], n[4], n[5], 0, loc)
	if t.Month() != time.Month(n[1]) || t.Day() != n[2] || t.Hour() != n[3] || n[4] > 59 || n[5] > 59 {
		return time.Time{}, bad
	}
	return t, nil
}

// parseWindow parses start and end and requires end after start.
func parseWindow(startRaw, endRaw, timezone string) (time.Time, time.Time, error) {
	start, err := parseTime(startRaw, timezone)
	if err != nil {
		return start, start, err
	}
	end, err := parseTime(endRaw, timezone)
	if err != nil {
		return start, end, err
	}
	if !end.After(start) {
		return start, end, errs.Usage("event end must be after event start")
	}
	return start, end, nil
}

func isoformat(t time.Time) string { return t.Format("2006-01-02T15:04:05-07:00") }

func parseInvitees(value string) ([]string, error) {
	out := []string{}
	for _, item := range strings.Split(value, ",") {
		if s := strings.TrimSpace(item); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil, errs.Usage("invitees_csv must include at least one email")
	}
	return out, nil
}

// formatEventTime shows a timed event in timezone and an all-day event
// as its raw dates.
func formatEventTime(start, end, timezone string) string {
	if !strings.Contains(start, "T") || !strings.Contains(end, "T") {
		return start + " -> " + end
	}
	loc, err := config.Location(timezone)
	if err != nil {
		loc = time.UTC
	}
	return localStamp(start, loc) + " -> " + localStamp(end, loc)
}

func localStamp(value string, loc *time.Location) string {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t, err = time.ParseInLocation("2006-01-02T15:04:05", value, time.Local)
	}
	if err != nil {
		return value
	}
	return t.In(loc).Format("2006-01-02 15:04:05")
}

func eventRow(ev calendar.Event, timezone string) map[string]any {
	return map[string]any{
		"event_id":    ev.ID,
		"title":       ev.Title("(untitled)"),
		"time":        formatEventTime(ev.Start.Value(), ev.End.Value(), timezone),
		"attendees":   ev.AttendeeEmails(),
		"meeting_url": ev.MeetingURL(),
	}
}

func printEvents(w io.Writer, events []calendar.Event, timezone string, asJSON bool) {
	if asJSON {
		rows := make([]any, len(events))
		for i, ev := range events {
			rows[i] = eventRow(ev, timezone)
		}
		printJSON(w, map[string]any{"events": rows})
		return
	}
	if len(events) == 0 {
		fmt.Fprintln(w, "no events")
		return
	}
	sections := make([]string, len(events))
	for i, ev := range events {
		sections[i] = eventSection(i+1, ev, timezone)
	}
	fmt.Fprintln(w, strings.Join(sections, "\n"))
}

func eventSection(index int, ev calendar.Event, timezone string) string {
	prefix := fmt.Sprintf("[%d]", index)
	attendees := strings.Join(ev.AttendeeEmails(), ",")
	if attendees == "" {
		attendees = "-"
	}
	lines := []string{
		prefix + strings.Repeat("-", max(1, 79-len(prefix))),
		fmt.Sprintf("%-9s : %s", "event_id", ev.ID),
		fmt.Sprintf("%-9s : %s", "title", ev.Title("(untitled)")),
		fmt.Sprintf("%-9s : %s", "time", formatEventTime(ev.Start.Value(), ev.End.Value(), timezone)),
		fmt.Sprintf("%-9s : %s", "attendees", attendees),
		ev.MeetingURL(),
	}
	return strings.Join(lines, "\n")
}
