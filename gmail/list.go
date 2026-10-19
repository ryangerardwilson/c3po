package gmail

import (
	"fmt"
	"strings"

	"github.com/ryangerardwilson/c3po/gmail/internal/api"
	"github.com/ryangerardwilson/c3po/gmail/internal/config"
	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
)

func (e Env) list(app *config.App, acct config.Account, params []string) (int, error) {
	q, err := parseList(params)
	if err != nil {
		return 0, err
	}
	if q.Audit != "" {
		if q.Open {
			return 0, errs.Usage("ls -o is not supported with -ura or -ra")
		}
		if q.Attachments {
			return 0, errs.Usage("ls -wa is not supported with -ura or -ra")
		}
		return e.audit(app, acct, q)
	}
	if q.Thread == "" && len(q.Terms) == 0 && !q.Attachments {
		return 0, errs.Usage("%s", listShape)
	}
	svc, err := e.service(acct)
	if err != nil {
		return 0, err
	}
	var messages []api.Message
	if q.Thread != "" {
		messages, err = svc.Thread(q.Thread)
	} else {
		messages, err = svc.List(gmailQuery(q, acct.Email), listLimit(q, app.DefaultListLimit))
	}
	if err != nil {
		return 0, err
	}
	if q.Attachments {
		messages = withFiles(messages)
	}
	messages = sortByInternal(messages)
	e.printList(svc, messages, app.TimezoneOffset, q.Open, q.JSON)
	return 0, nil
}

func gmailQuery(q listQuery, email string) string {
	mode, rest := splitMode(q.Terms)
	terms := modeTerms(mode, email)
	terms = append(terms, rest...)
	if q.Attachments {
		terms = append(terms, "has:attachment")
	}
	return strings.Join(terms, " ")
}

func splitMode(terms []string) (string, []string) {
	if len(terms) == 0 {
		return "", nil
	}
	if word(terms[0], "unread", "read", "starred", "external", "sent") {
		return terms[0], terms[1:]
	}
	return "", terms
}

func modeTerms(mode, email string) []string {
	switch mode {
	case "unread":
		return []string{"is:unread", "-in:sent"}
	case "read":
		return []string{"is:read", "-in:sent"}
	case "starred":
		return []string{"is:starred", "-in:sent"}
	case "sent":
		return []string{"in:sent"}
	case "external":
		terms := []string{"-from:" + email, "-in:sent"}
		if d := domainOf(email); d != "" {
			terms = append(terms, "-from:*@"+d)
		}
		return terms
	default:
		return []string{"-in:sent"}
	}
}

func listLimit(q listQuery, fallback int) int {
	if q.Limit != nil {
		return *q.Limit
	}
	mode, rest := splitMode(q.Terms)
	if len(rest) > 0 || (q.Attachments && mode == "") || (q.Attachments && mode == "sent" && len(rest) == 0) {
		return fallback
	}
	if mode == "sent" && len(rest) == 0 && !q.Attachments {
		return 0
	}
	return 0
}

func withFiles(messages []api.Message) []api.Message {
	var out []api.Message
	for _, msg := range messages {
		if msg.HasFile {
			out = append(out, msg)
		}
	}
	return out
}

func (e Env) printList(svc Service, messages []api.Message, offset string, open, asJSON bool) {
	if asJSON {
		rows := make([]map[string]any, 0, len(messages))
		for _, msg := range messages {
			rows = append(rows, summaryMap(summarize(msg, offset, true)))
		}
		e.printJSON(map[string]any{"messages": rows})
		return
	}
	if !open {
		fmt.Fprintln(e.Stdout, renderTable(messages, offset))
		return
	}
	if len(messages) == 0 {
		fmt.Fprintln(e.Stdout, "No messages found.")
		return
	}
	var ids []string
	for i, msg := range messages {
		if msg.ID != "" {
			ids = append(ids, msg.ID)
		}
		fmt.Fprintf(e.Stdout, "[%d/%d]\n", i+1, len(messages))
		fmt.Fprintln(e.Stdout, renderOpen(msg, offset))
		if i < len(messages)-1 {
			fmt.Fprintln(e.Stdout)
		}
	}
	n, _ := svc.BatchModify(ids, nil, []string{"UNREAD"})
	fmt.Fprintf(e.Stdout, "ls_opened messages=%d marked_read=%d\n", len(messages), n)
}

func domainOf(email string) string {
	_, addr, ok := strings.Cut(email, "@")
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(addr))
}
