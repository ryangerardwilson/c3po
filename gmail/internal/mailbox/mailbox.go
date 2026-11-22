// Package mailbox is the Gmail message as the c3po gmail commands see it.
// The REST client translates Gmail responses into these types.
package mailbox

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
