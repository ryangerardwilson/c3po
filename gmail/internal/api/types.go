// Package api is the Gmail REST client used by c3po gmail.
package api

// Attachment is one downloadable file part.
type Attachment struct {
	Filename string
	ID       string
	Mime     string
}

// Message is one Gmail message the commands print or modify.
type Message struct {
	ID          string
	ThreadID    string
	History     string
	Snippet     string
	Labels      []string
	Headers     map[string]string
	Plain       string
	HTML        string
	Raw         []byte
	Internal    int64
	HasFile     bool
	Attachments []Attachment
}

// SenderCount is one From address and how many unread messages it has.
type SenderCount struct {
	Sender string
	Count  int
}
