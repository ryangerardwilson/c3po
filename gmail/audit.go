package gmail

import (
	"fmt"
	"strings"

	"github.com/ryangerardwilson/c3po/gmail/internal/api"
	"github.com/ryangerardwilson/c3po/gmail/internal/config"
)

func (e Env) audit(app *config.App, acct config.Account, q listQuery) (int, error) {
	flag := "ura"
	query := "is:unread -in:sent"
	if q.Audit == "read" {
		flag = "ra"
		query = "is:read -in:sent"
	}
	fmt.Fprintln(e.Stdout, "Audit mode: enter 's' for spam (add sender to spam_senders + trash message), 't' for trash only, 'n' for not spam (leave unchanged), 'q' to stop.")
	if e.Ask == nil && !interactive(e.Stdin) {
		fmt.Fprintf(e.Stdout, "ls -%s complete: audited=0 trashed=0\n", flag)
		return 0, nil
	}
	svc, err := e.service(acct)
	if err != nil {
		return 0, err
	}
	limit := 0
	if q.Limit != nil {
		limit = *q.Limit
	} else {
		limit = app.DefaultListLimit
	}
	messages, err := svc.List(query, limit)
	if err != nil {
		return 0, err
	}
	messages = sortByInternal(messages)
	if len(messages) == 0 {
		fmt.Fprintf(e.Stdout, "No %s messages found.\n", q.Audit)
		return 0, nil
	}
	audited, trashed, senders, err := e.auditBatch(svc, app, acct, messages)
	if err != nil {
		return 0, err
	}
	if len(senders) > 0 {
		if err := config.UpdateSpamSenders(app.Path, acct.Preset, senders); err != nil {
			return 0, err
		}
	}
	fmt.Fprintf(e.Stdout, "ls -%s complete: audited=%d trashed=%d\n", flag, audited, trashed)
	return 0, nil
}

func (e Env) auditBatch(svc Service, app *config.App, acct config.Account, messages []api.Message) (int, int, []string, error) {
	senders := append([]string{}, acct.SpamSenders...)
	seen := map[string]bool{}
	for _, s := range senders {
		seen[strings.ToLower(s)] = true
	}
	audited, trashed := 0, 0
	for _, msg := range messages {
		fmt.Fprintln(e.Stdout, renderOpen(msg, app.TimezoneOffset))
		answer, err := e.ask("audit [s/t/n/q]: ")
		if err != nil {
			return audited, trashed, senders, err
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "q", "":
			return audited, trashed, senders, nil
		case "n":
			audited++
		case "t":
			if err := svc.TrashOne(msg.ID); err != nil {
				return audited, trashed, senders, err
			}
			audited++
			trashed++
		case "s":
			sender := strings.ToLower(summarize(msg, app.TimezoneOffset, true).FromEmail)
			if sender != "" && !seen[sender] {
				seen[sender] = true
				senders = append(senders, sender)
			}
			if err := svc.TrashOne(msg.ID); err != nil {
				return audited, trashed, senders, err
			}
			audited++
			trashed++
		default:
			audited++
		}
	}
	return audited, trashed, senders, nil
}
