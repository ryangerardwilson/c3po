package gmail

import (
	"fmt"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
)

type envelope struct {
	From, To, Subject, Body string
	CC, BCC                 []string
	Files                   []string
	InReplyTo, Refs         string
}

func buildMIME(m envelope) ([]byte, error) {
	files, err := flattenFiles(m.Files)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return []byte(plainMessage(m)), nil
	}
	return mixedMessage(m, files)
}

func plainMessage(m envelope) string {
	var b strings.Builder
	writeHeaders(&b, m)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(strings.ReplaceAll(m.Body, "\n", "\r\n"))
	return b.String()
}

func writeHeaders(b *strings.Builder, m envelope) {
	fmt.Fprintf(b, "To: %s\r\n", m.To)
	fmt.Fprintf(b, "From: %s\r\n", m.From)
	fmt.Fprintf(b, "Subject: %s\r\n", m.Subject)
	if len(m.CC) > 0 {
		fmt.Fprintf(b, "Cc: %s\r\n", strings.Join(m.CC, ", "))
	}
	if len(m.BCC) > 0 {
		fmt.Fprintf(b, "Bcc: %s\r\n", strings.Join(m.BCC, ", "))
	}
	if m.InReplyTo != "" {
		fmt.Fprintf(b, "In-Reply-To: %s\r\n", m.InReplyTo)
	}
	if m.Refs != "" {
		fmt.Fprintf(b, "References: %s\r\n", m.Refs)
	}
}

func mixedMessage(m envelope, files []string) ([]byte, error) {
	var b strings.Builder
	mp := multipart.NewWriter(&b)
	writeHeaders(&b, m)
	fmt.Fprintf(&b, "MIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%s\r\n\r\n", mp.Boundary())
	h := textproto.MIMEHeader{}
	h.Set("Content-Type", `text/plain; charset="utf-8"`)
	h.Set("Content-Transfer-Encoding", "8bit")
	w, err := mp.CreatePart(h)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write([]byte(strings.ReplaceAll(m.Body, "\n", "\r\n"))); err != nil {
		return nil, err
	}
	for _, path := range files {
		if err := addFile(mp, path); err != nil {
			return nil, err
		}
	}
	if err := mp.Close(); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

func addFile(mp *multipart.Writer, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	kind := mime.TypeByExtension(filepath.Ext(path))
	if kind == "" {
		kind = "application/octet-stream"
	}
	h := textproto.MIMEHeader{}
	h.Set("Content-Type", kind)
	h.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(path)))
	w, err := mp.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func flattenFiles(paths []string) ([]string, error) {
	var out []string
	for _, path := range paths {
		st, err := os.Stat(path)
		if err != nil {
			return nil, errs.Usage("-atch path not found: %s", path)
		}
		if st.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					out = append(out, filepath.Join(path, entry.Name()))
				}
			}
			continue
		}
		if !st.Mode().IsRegular() {
			return nil, errs.Usage("-atch must point to a file or directory: %s", path)
		}
		out = append(out, path)
	}
	return out, nil
}

func replySubject(subject string) string {
	if strings.HasPrefix(strings.ToLower(subject), "re:") {
		return subject
	}
	if subject == "" {
		return "Re:"
	}
	return "Re: " + subject
}

func replyRecipients(headers map[string]string, from string, all bool) (string, []string) {
	mine := strings.ToLower(strings.TrimSpace(from))
	source := headerOf(headers, "reply-to")
	if source == "" {
		source = headerOf(headers, "from")
	}
	to := firstOther(source, mine)
	if to == "" || !all {
		return to, nil
	}
	return to, normalizeRecipients(addresses(headerOf(headers, "cc")), mine, []string{to})
}

func firstOther(value, mine string) string {
	for _, addr := range addresses(value) {
		if !strings.EqualFold(addr, mine) {
			return addr
		}
	}
	return ""
}

func addresses(value string) []string {
	list, err := mail.ParseAddressList(value)
	if err != nil {
		if addr, err := mail.ParseAddress(value); err == nil {
			return []string{addr.Address}
		}
		return nil
	}
	out := make([]string, 0, len(list))
	for _, addr := range list {
		out = append(out, addr.Address)
	}
	return out
}

func normalizeRecipients(values []string, mine string, exclude []string) []string {
	denied := map[string]bool{strings.ToLower(mine): true}
	for _, item := range exclude {
		denied[strings.ToLower(item)] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, value := range values {
		for _, addr := range addresses(value) {
			key := strings.ToLower(addr)
			if key == "" || denied[key] || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, addr)
		}
	}
	return out
}

func headerOf(headers map[string]string, name string) string {
	if headers == nil {
		return ""
	}
	return headers[strings.ToLower(name)]
}

func joinRefs(refs, id string) string {
	refs = strings.TrimSpace(refs)
	id = strings.TrimSpace(id)
	if id == "" {
		return refs
	}
	if refs == "" {
		return id
	}
	return refs + " " + id
}
