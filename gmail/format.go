package gmail

import (
	"fmt"
	"html"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/gmail/internal/api"
)

type summary struct {
	ID, Thread, From, FromEmail, To, CC, BCC, Subject, Date, Body, Snippet string
}

func summarize(msg api.Message, offset string, trim bool) summary {
	fromName, fromEmail := splitAddr(header(msg, "from"))
	display := fromEmail
	if fromName != "" && fromEmail != "" {
		display = fromName + " <" + fromEmail + ">"
	} else if display == "" {
		display = header(msg, "from")
	}
	body := strings.TrimSpace(msg.Plain)
	if body == "" {
		body = htmlToText(msg.HTML)
	}
	body = stripHistory(body)
	if trim {
		body = trimBody(body, 24, 2000)
	}
	snippet := strings.TrimSpace(msg.Snippet)
	if body == "" {
		body = snippet
	}
	return summary{
		ID: msg.ID, Thread: msg.ThreadID, From: display, FromEmail: strings.ToLower(fromEmail),
		To: header(msg, "to"), CC: header(msg, "cc"), BCC: header(msg, "bcc"),
		Subject: header(msg, "subject"), Date: localDate(header(msg, "date"), offset),
		Body: strings.TrimSpace(body), Snippet: snippet,
	}
}

func summaryMap(s summary) map[string]any {
	return map[string]any{
		"message_id": s.ID, "thread_id": s.Thread, "from": s.From, "from_email": s.FromEmail,
		"to": s.To, "cc": s.CC, "bcc": s.BCC, "subject": s.Subject, "date": s.Date,
		"body": s.Body, "snippet": s.Snippet,
	}
}

func renderTable(messages []api.Message, offset string) string {
	if len(messages) == 0 {
		return "No messages found."
	}
	var sections []string
	for i, msg := range messages {
		row := summarize(msg, offset, true)
		prefix := fmt.Sprintf("[%d]", i+1)
		n := 79 - len(prefix)
		if n < 1 {
			n = 1
		}
		lines := []string{
			prefix + strings.Repeat("-", n),
			"message_id: " + row.ID,
			"thread_id : " + row.Thread,
			"date      : " + row.Date,
			"from      : " + row.From,
			"subject   : " + row.Subject,
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}
	return strings.Join(sections, "\n")
}

func renderOpen(msg api.Message, offset string) string {
	row := summarize(msg, offset, false)
	lines := []string{
		"message_id: " + row.ID,
		"thread_id : " + row.Thread,
		"date      : " + row.Date,
		"from      : " + row.From,
		"to        : " + row.To,
	}
	if strings.TrimSpace(row.CC) != "" {
		lines = append(lines, "cc        : "+row.CC)
	}
	if strings.TrimSpace(row.BCC) != "" {
		lines = append(lines, "bcc       : "+row.BCC)
	}
	lines = append(lines, "subject   : "+row.Subject, "body:", "", row.Body, "")
	return strings.Join(lines, "\n")
}

func header(msg api.Message, name string) string {
	if msg.Headers == nil {
		return ""
	}
	return msg.Headers[strings.ToLower(name)]
}

func splitAddr(raw string) (string, string) {
	addr, err := mail.ParseAddress(raw)
	if err != nil {
		return "", emailOnly(raw)
	}
	return addr.Name, addr.Address
}

func emailOnly(raw string) string {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "@"); i < 0 {
		return ""
	}
	return strings.Trim(raw, "<> ")
}

func localDate(raw, offset string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	t, err := mail.ParseDate(raw)
	if err != nil {
		return raw
	}
	return t.In(offsetZone(offset)).Format("2006-01-02 15:04:05")
}

func offsetZone(offset string) *time.Location {
	if len(offset) != 6 {
		return time.FixedZone("local", 0)
	}
	sign := 1
	if offset[0] == '-' {
		sign = -1
	}
	hh, _ := strconv.Atoi(offset[1:3])
	mm, _ := strconv.Atoi(offset[4:6])
	return time.FixedZone("local", sign*(hh*3600+mm*60))
}

var (
	tagRe = regexp.MustCompile(`(?is)<[^>]+>`)
	onRe  = regexp.MustCompile(`(?i)^on .+ wrote:?\s*$`)
)

func htmlToText(s string) string {
	s = tagRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(html.UnescapeString(s))
}

func stripHistory(body string) string {
	if body == "" {
		return ""
	}
	var kept []string
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, ">") || onRe.MatchString(trim) {
			break
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

func trimBody(body string, maxLines, maxChars int) string {
	if len(body) > maxChars {
		body = strings.TrimRight(body[:maxChars], " \t\r\n")
	}
	lines := strings.Split(body, "\n")
	if len(lines) > maxLines {
		body = strings.TrimRight(strings.Join(lines[:maxLines], "\n"), " \t\r\n")
	}
	return body
}

func sortByInternal(messages []api.Message) []api.Message {
	out := append([]api.Message(nil), messages...)
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j].Internal < out[j-1].Internal {
			out[j], out[j-1] = out[j-1], out[j]
			j--
		}
	}
	return out
}
