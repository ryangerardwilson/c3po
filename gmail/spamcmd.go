package gmail

import (
	"fmt"
	"strings"

	"github.com/ryangerardwilson/c3po/gmail/internal/api"
	"github.com/ryangerardwilson/c3po/gmail/internal/config"
	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
	"github.com/ryangerardwilson/c3po/gmail/internal/spam"
)

func (e Env) spamCleanAll() (int, error) {
	app, err := e.load()
	if err != nil {
		return 0, err
	}
	if len(app.Presets) == 0 {
		return 0, errs.Usage("No configured presets found.")
	}
	total := 0
	for _, preset := range app.Presets {
		acct := app.Accounts[preset]
		n, err := e.cleanAccount(acct, true)
		if err != nil {
			return 0, err
		}
		total += n
		fmt.Fprintf(e.Stdout, "%s\ttrashed_spam=%d\n", preset, n)
	}
	fmt.Fprintf(e.Stdout, "spam clean complete: presets=%d trashed_spam=%d\n", len(app.Presets), total)
	return 0, nil
}

func (e Env) spamPreset(app *config.App, acct config.Account, params []string) (int, error) {
	if len(params) == 0 {
		return 0, errs.Usage("Use: c3po gmail <preset> spam inspect|clean|add|allow")
	}
	switch params[0] {
	case "inspect":
		if len(params) != 1 {
			return 0, errs.Usage("Use: c3po gmail <preset> spam inspect")
		}
		return e.spamInspect(app, acct)
	case "clean":
		if len(params) != 1 {
			return 0, errs.Usage("Use: c3po gmail <preset> spam clean")
		}
		n, err := e.cleanAccount(acct, false)
		if err != nil {
			return 0, err
		}
		fmt.Fprintf(e.Stdout, "trashed_spam=%d\nspam clean complete: trashed_spam=%d\n", n, n)
		return 0, nil
	case "add":
		return e.spamAdd(app, acct, params[1:])
	case "allow":
		return e.spamAllow(app, acct, params[1:])
	case "mark":
		if len(params) == 3 && params[1] == "message" {
			return e.spamMark(app, acct, params[2])
		}
	}
	return 0, errs.Usage("Use: c3po gmail <preset> spam inspect|clean|add|allow")
}

func (e Env) cleanAccount(acct config.Account, banner bool) (int, error) {
	if banner {
		fmt.Fprintf(e.Stdout, "preset=%s email=%s\n", acct.Preset, acct.Email)
	}
	svc, err := e.service(acct)
	if err != nil {
		return 0, err
	}
	return cleanup(svc, acct, e.Stdout)
}

func cleanup(svc Service, acct config.Account, out interface{ Write([]byte) (int, error) }) (int, error) {
	senders := spam.CleanupSenders(acct.SpamSenders, acct.SpamExcludes)
	groups := spam.Groups(senders, 25)
	if len(groups) == 0 {
		return 0, nil
	}
	fmt.Fprintf(out, "spam sender groups to process: %d\n", len(groups))
	var all []string
	for i, group := range groups {
		ids, err := svc.ListIDs(spam.Query(group))
		if err != nil {
			return 0, err
		}
		all = append(all, ids...)
		fmt.Fprintf(out, "processed group %d/%d: ids_found=%d unique_ids_collected=%d\n", i+1, len(groups), len(ids), len(api.SortedIDs(all)))
	}
	n, err := svc.TrashIDs(api.SortedIDs(all))
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(out, "trashed so far: %d\n", n)
	return n, nil
}

func (e Env) spamInspect(app *config.App, acct config.Account) (int, error) {
	fmt.Fprintln(e.Stdout, "scanning unread non-gmail sender sample...")
	svc, err := e.service(acct)
	if err != nil {
		return 0, err
	}
	counts, err := svc.UnreadSenders(func(event string, value, total int) {
		switch event {
		case "listed_unread_messages":
			fmt.Fprintf(e.Stdout, "unread non-gmail messages listed: %d\n", value)
		case "counted_sender_messages":
			fmt.Fprintf(e.Stdout, "counting sender occurrences: %d/%d\n", value, total)
		case "unique_senders":
			fmt.Fprintf(e.Stdout, "unique non-gmail senders found: %d\n", value)
		}
	})
	if err != nil {
		return 0, err
	}
	found := spam.Candidates(counts, acct.SpamSenders, acct.SpamExcludes, acct.Email, 5)
	fmt.Fprintf(e.Stdout, "potential spammers identified: %d\n", len(found))
	if len(found) == 0 {
		fmt.Fprintln(e.Stdout, "no potential spam senders found")
		return 0, nil
	}
	fmt.Fprintln(e.Stdout, "potential spam senders (>5 unread, excluding gmail + preset domain + spam_excludes):")
	var add []string
	for i, item := range found {
		fmt.Fprintf(e.Stdout, "  %d. %s (unread=%d)\n", i+1, item.Sender, item.Unread)
		add = append(add, item.Sender)
	}
	fmt.Fprintf(e.Stdout, "review: add_to_spam=%d\n", len(add))
	answer, err := e.ask("confirm update config? [y/N]: ")
	if err != nil {
		return 0, err
	}
	if strings.ToLower(strings.TrimSpace(answer)) != "y" {
		fmt.Fprintln(e.Stdout, "skipped by user")
		return 0, nil
	}
	merged := mergeUnique(acct.SpamSenders, add)
	if err := config.UpdateSpamSenders(app.Path, acct.Preset, merged); err != nil {
		return 0, err
	}
	fmt.Fprintf(e.Stdout, "updated: +%d spam\nspam inspect complete: spam_added=%d\n", len(add), len(add))
	return 0, nil
}

func (e Env) spamAdd(app *config.App, acct config.Account, params []string) (int, error) {
	if len(params) != 1 {
		return 0, errs.Usage("Use: c3po gmail <preset> spam add <email_or_domain_csv>|unread")
	}
	if params[0] == "unread" {
		return e.spamAddUnread(app, acct)
	}
	items := config.NormalizeSpamSenderList(csv(params[0]))
	if len(items) == 0 {
		return 0, errs.Usage("spam add requires at least one valid email in comma-separated input")
	}
	merged := mergeUnique(acct.SpamSenders, items)
	if err := config.UpdateSpamSenders(app.Path, acct.Preset, merged); err != nil {
		return 0, err
	}
	fmt.Fprintf(e.Stdout, "spam add complete: added=%d total_spam_senders=%d\n", len(items), len(merged))
	return 0, nil
}

func (e Env) spamAddUnread(app *config.App, acct config.Account) (int, error) {
	fmt.Fprintln(e.Stdout, "spam add unread: scanning unread messages in batches...")
	svc, err := e.service(acct)
	if err != nil {
		return 0, err
	}
	page := ""
	senders := map[string]bool{}
	var ids []string
	batch := 0
	for {
		msgs, next, err := svc.ListPage("is:unread", 100, page)
		if err != nil {
			return 0, err
		}
		if len(msgs) == 0 {
			break
		}
		batch++
		for _, msg := range msgs {
			sender := strings.ToLower(strings.TrimSpace(summarize(msg, app.TimezoneOffset, true).FromEmail))
			if sender != "" {
				senders[sender] = true
			}
			if msg.ID != "" {
				ids = append(ids, msg.ID)
			}
		}
		fmt.Fprintf(e.Stdout, "processed unread batch %d: senders_collected=%d messages_collected=%d\n", batch, len(senders), len(ids))
		if next == "" {
			break
		}
		page = next
	}
	if len(ids) == 0 {
		fmt.Fprintln(e.Stdout, "spam add unread complete: no unread messages found")
		return 0, nil
	}
	var list []string
	for sender := range senders {
		list = append(list, sender)
	}
	items := config.NormalizeSpamSenderList(list)
	merged := mergeUnique(acct.SpamSenders, items)
	if err := config.UpdateSpamSenders(app.Path, acct.Preset, merged); err != nil {
		return 0, err
	}
	n, err := svc.TrashIDs(ids)
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(e.Stdout, "spam add unread complete: added_senders=%d total_spam_senders=%d trashed_unread=%d\n", len(items), len(merged), n)
	return 0, nil
}

func (e Env) spamAllow(app *config.App, acct config.Account, params []string) (int, error) {
	if len(params) != 1 {
		return 0, errs.Usage("Use: c3po gmail <preset> spam allow <email_or_domain_csv>")
	}
	items := config.NormalizeSenderList(csv(params[0]))
	if len(items) == 0 {
		return 0, errs.Usage("spam allow requires at least one valid email in comma-separated input")
	}
	merged := mergeUnique(acct.SpamExcludes, items)
	if err := config.UpdateSpamExcludes(app.Path, acct.Preset, merged); err != nil {
		return 0, err
	}
	fmt.Fprintf(e.Stdout, "spam allow complete: added=%d total_spam_excludes=%d\n", len(items), len(merged))
	return 0, nil
}

func (e Env) spamMark(app *config.App, acct config.Account, id string) (int, error) {
	svc, err := e.service(acct)
	if err != nil {
		return 0, err
	}
	msg, err := svc.Meta(id)
	if err != nil {
		return 0, err
	}
	sender := strings.ToLower(strings.TrimSpace(summarize(msg, app.TimezoneOffset, true).FromEmail))
	if sender != "" {
		merged := mergeUnique(acct.SpamSenders, []string{sender})
		if err := config.UpdateSpamSenders(app.Path, acct.Preset, merged); err != nil {
			return 0, err
		}
	}
	if err := svc.TrashOne(id); err != nil {
		return 0, err
	}
	if sender != "" {
		fmt.Fprintf(e.Stdout, "marked_spammer sender=%s trashed_message_id=%s\n", sender, id)
	} else {
		fmt.Fprintf(e.Stdout, "trashed_message_id=%s (sender not detected)\n", id)
	}
	return 0, nil
}

func mergeUnique(existing, incoming []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range append(append([]string{}, existing...), incoming...) {
		key := strings.ToLower(strings.TrimSpace(item))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}
