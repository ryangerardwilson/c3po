package api

import (
	"encoding/base64"
	"encoding/json"
	"net/mail"
	"strings"
)

func parseMessage(raw []byte) (Message, error) {
	var body struct {
		ID       string   `json:"id"`
		ThreadID string   `json:"threadId"`
		History  string   `json:"historyId"`
		Snippet  string   `json:"snippet"`
		Labels   []string `json:"labelIds"`
		Internal string   `json:"internalDate"`
		Payload  part     `json:"payload"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return Message{}, err
	}
	msg := Message{
		ID: body.ID, ThreadID: body.ThreadID, History: body.History,
		Snippet: body.Snippet, Labels: body.Labels, Headers: map[string]string{},
	}
	if n, err := parseInt(body.Internal); err == nil {
		msg.Internal = n
	}
	walkPart(&msg, body.Payload)
	return msg, nil
}

type part struct {
	Mime    string `json:"mimeType"`
	Headers []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"headers"`
	Body struct {
		Data         string `json:"data"`
		AttachmentID string `json:"attachmentId"`
		Size         int    `json:"size"`
	} `json:"body"`
	Filename string `json:"filename"`
	Parts    []part `json:"parts"`
}

func walkPart(msg *Message, p part) {
	for _, h := range p.Headers {
		key := strings.ToLower(h.Name)
		if _, ok := msg.Headers[key]; !ok {
			msg.Headers[key] = h.Value
		}
	}
	if p.Filename != "" && !skipAttachment(p.Filename, p.Mime) {
		msg.HasFile = true
		msg.Attachments = append(msg.Attachments, Attachment{Filename: p.Filename, ID: p.Body.AttachmentID, Mime: p.Mime})
	}
	text := decodePart(p.Body.Data)
	switch p.Mime {
	case "text/plain":
		if msg.Plain == "" {
			msg.Plain = text
		}
	case "text/html":
		if msg.HTML == "" {
			msg.HTML = text
		}
	}
	for _, child := range p.Parts {
		walkPart(msg, child)
	}
}

func skipAttachment(name, mime string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".ics") || mime == "text/calendar"
}

func decodePart(data string) string {
	if data == "" {
		return ""
	}
	raw, err := decodeRawURL(data)
	if err != nil {
		return ""
	}
	return string(raw)
}

func decodeRawURL(data string) ([]byte, error) {
	data = strings.TrimSpace(data)
	if m := len(data) % 4; m != 0 {
		data += strings.Repeat("=", 4-m)
	}
	return base64.URLEncoding.DecodeString(data)
}

func rawURL(raw []byte) string {
	return base64.RawURLEncoding.EncodeToString(raw)
}

func parseInt(s string) (int64, error) {
	return json.Number(s).Int64()
}

func header(msg Message, name string) string {
	return msg.Headers[strings.ToLower(name)]
}

func emailOf(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	addr, err := mail.ParseAddress(value)
	if err != nil {
		if strings.Contains(value, "@") {
			return strings.ToLower(strings.Trim(value, "<> "))
		}
		return ""
	}
	return strings.ToLower(addr.Address)
}
