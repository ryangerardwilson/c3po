// Package transcripts finds a Meet transcript attached to an event and
// saves its exported text.
package transcripts

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/gcal/internal/calendar"
	"github.com/ryangerardwilson/c3po/gcal/internal/errs"
)

// GoogleDocMIME is the Drive MIME type of a Google Doc.
const GoogleDocMIME = "application/vnd.google-apps.document"

// Attachment is the transcript Doc of an event.
type Attachment struct {
	FileID string
	Title  string
}

func looksLikeTranscript(a calendar.Attachment) bool {
	title := strings.ToLower(strings.TrimSpace(a.Title))
	if !strings.Contains(title, "transcript") {
		return false
	}
	if strings.TrimSpace(a.MimeType) == GoogleDocMIME {
		return true
	}
	return strings.Contains(strings.TrimSpace(a.FileURL), "docs.google.com/document")
}

// Find returns the first transcript-like attachment with a file id.
func Find(ev calendar.Event) (Attachment, error) {
	for _, a := range ev.Attachments {
		if !looksLikeTranscript(a) {
			continue
		}
		id := strings.TrimSpace(a.FileID)
		title := strings.TrimSpace(a.Title)
		if title == "" {
			title = "Transcript"
		}
		if id != "" {
			return Attachment{FileID: id, Title: title}, nil
		}
	}
	return Attachment{}, errs.API("No transcript attachment found for this event.")
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

// Slug is the title reduced to lowercase words joined by dashes.
func Slug(title string) string {
	s := strings.ToLower(strings.Trim(nonAlnum.ReplaceAllString(strings.TrimSpace(title), "-"), "-"))
	if s == "" {
		return "meeting"
	}
	return s
}

// StartFragment is the event start date used in the file name.
func StartFragment(start string) string {
	raw := strings.TrimSpace(start)
	switch {
	case raw == "":
		return "unknown-date"
	case !strings.Contains(raw, "T"):
		return strings.ReplaceAll(raw, "/", "-")
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006-01-02")
		}
	}
	before, _, _ := strings.Cut(raw, "T")
	return before
}

// Save writes text as <date>_<slug>_<event id>.txt under dir.
func Save(dir, text, eventID, title, start string) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, StartFragment(start)+"_"+Slug(title)+"_"+eventID+".txt")
	return path, os.WriteFile(path, []byte(text), 0o644)
}
