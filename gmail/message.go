package gmail

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ryangerardwilson/c3po/gmail/internal/api"
	"github.com/ryangerardwilson/c3po/gmail/internal/config"
	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
)

const inspectShape = "Use: c3po gmail <preset> inspect message <id> | inspect thread <id> [output json]"
const openShape = "Use: c3po gmail <preset> open message <id> | open thread <id>"

func (e Env) open(acct config.Account, params []string, offset string) (int, error) {
	if len(params) != 2 || (params[0] != "message" && params[0] != "thread") {
		return 0, errs.Usage("%s", openShape)
	}
	svc, err := e.service(acct)
	if err != nil {
		return 0, err
	}
	if params[0] == "message" {
		return e.openMessage(svc, acct, params[1], offset)
	}
	return e.openThread(svc, acct, params[1], offset)
}

func (e Env) openMessage(svc Service, acct config.Account, id, offset string) (int, error) {
	msg, err := svc.Get(id)
	if err != nil {
		return 0, err
	}
	fmt.Fprintln(e.Stdout, renderOpen(msg, offset))
	n, err := e.saveAttachments(svc, acct.Preset, msg)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		fmt.Fprintf(e.Stdout, "attachments_downloaded=%d dir=%s\n", n, e.attachDir(acct.Preset, msg.ID))
	}
	mid, thread, err := svc.Modify(id, nil, []string{"UNREAD"})
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(e.Stdout, "opened_and_marked_read message_id=%s thread_id=%s\n", mid, thread)
	return 0, nil
}

func (e Env) openThread(svc Service, acct config.Account, id, offset string) (int, error) {
	messages, err := svc.Thread(id)
	if err != nil {
		return 0, err
	}
	messages = sortByInternal(messages)
	if len(messages) == 0 {
		fmt.Fprintf(e.Stdout, "no messages found in thread_id=%s\n", id)
		return 0, nil
	}
	var ids []string
	downloaded := 0
	dirs := 0
	for i, msg := range messages {
		fmt.Fprintf(e.Stdout, "[%d/%d]\n", i+1, len(messages))
		fmt.Fprintln(e.Stdout, renderOpen(msg, offset))
		if i < len(messages)-1 {
			fmt.Fprintln(e.Stdout)
		}
		if msg.ID != "" {
			ids = append(ids, msg.ID)
		}
		n, err := e.saveAttachments(svc, acct.Preset, msg)
		if err != nil {
			return 0, err
		}
		if n > 0 {
			downloaded += n
			dirs++
		}
	}
	marked, err := svc.BatchModify(ids, nil, []string{"UNREAD"})
	if err != nil {
		return 0, err
	}
	if downloaded > 0 {
		fmt.Fprintf(e.Stdout, "attachments_downloaded=%d dirs=%d\n", downloaded, dirs)
	}
	fmt.Fprintf(e.Stdout, "opened_thread thread_id=%s messages=%d marked_read=%d\n", id, len(messages), marked)
	return 0, nil
}

func (e Env) inspect(acct config.Account, params []string, offset string) (int, error) {
	rest, asJSON, err := splitOutput(params, inspectShape)
	if err != nil {
		return 0, err
	}
	if len(rest) != 2 || (rest[0] != "message" && rest[0] != "thread") {
		return 0, errs.Usage("%s", inspectShape)
	}
	svc, err := e.service(acct)
	if err != nil {
		return 0, err
	}
	if rest[0] == "message" {
		return e.inspectMessage(svc, rest[1], offset, asJSON)
	}
	return e.inspectThread(svc, rest[1], offset, asJSON)
}

func (e Env) inspectMessage(svc Service, id, offset string, asJSON bool) (int, error) {
	msg, err := svc.Get(id)
	if err != nil {
		return 0, err
	}
	if asJSON {
		e.printJSON(map[string]any{"message": summaryMap(summarize(msg, offset, false))})
		return 0, nil
	}
	fmt.Fprintln(e.Stdout, renderOpen(msg, offset))
	fmt.Fprintf(e.Stdout, "inspected message_id=%s side_effects=none\n", id)
	return 0, nil
}

func (e Env) inspectThread(svc Service, id, offset string, asJSON bool) (int, error) {
	messages, err := svc.Thread(id)
	if err != nil {
		return 0, err
	}
	messages = sortByInternal(messages)
	if asJSON {
		rows := make([]map[string]any, 0, len(messages))
		for _, msg := range messages {
			rows = append(rows, summaryMap(summarize(msg, offset, false)))
		}
		e.printJSON(map[string]any{"thread_id": id, "messages": rows})
		return 0, nil
	}
	if len(messages) == 0 {
		fmt.Fprintf(e.Stdout, "no messages found in thread_id=%s\n", id)
		return 0, nil
	}
	for i, msg := range messages {
		fmt.Fprintf(e.Stdout, "[%d/%d]\n", i+1, len(messages))
		fmt.Fprintln(e.Stdout, renderOpen(msg, offset))
		if i < len(messages)-1 {
			fmt.Fprintln(e.Stdout)
		}
	}
	fmt.Fprintf(e.Stdout, "inspected_thread thread_id=%s messages=%d side_effects=none\n", id, len(messages))
	return 0, nil
}

func (e Env) mark(acct config.Account, params []string) (int, error) {
	if len(params) == 3 && params[0] == "all" && params[1] == "unread" && params[2] == "read" {
		return e.markAllRead(acct)
	}
	if len(params) != 3 || params[0] != "message" {
		return 0, errs.Usage("Use: c3po gmail <preset> mark message <id> read|unread|starred|unstarred")
	}
	svc, err := e.service(acct)
	if err != nil {
		return 0, err
	}
	id := params[1]
	var add, remove []string
	var label string
	switch params[2] {
	case "read":
		remove, label = []string{"UNREAD"}, "marked_read message_id=%s thread_id=%s\n"
	case "unread":
		add, label = []string{"UNREAD"}, "marked_unread message_id=%s thread_id=%s\n"
	case "starred":
		add, remove, label = []string{"STARRED"}, []string{"UNREAD"}, "starred message_id=%s thread_id=%s\n"
	case "unstarred":
		remove, label = []string{"STARRED"}, "unstarred message_id=%s thread_id=%s\n"
	default:
		return 0, errs.Usage("Use: c3po gmail <preset> mark message <id> read|unread|starred|unstarred")
	}
	mid, thread, err := svc.Modify(id, add, remove)
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(e.Stdout, label, mid, thread)
	return 0, nil
}

func (e Env) markAllRead(acct config.Account) (int, error) {
	svc, err := e.service(acct)
	if err != nil {
		return 0, err
	}
	ids, err := svc.ListIDs("is:unread")
	if err != nil {
		return 0, err
	}
	n, err := svc.BatchModify(ids, nil, []string{"UNREAD"})
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(e.Stdout, "marked_read_all=%d\n", n)
	return 0, nil
}

func (e Env) deletePreset(acct config.Account, params []string) (int, error) {
	svc, err := e.service(acct)
	if err != nil {
		return 0, err
	}
	if len(params) == 1 && params[0] == "unread" {
		ids, err := svc.ListIDs("is:unread -in:sent")
		if err != nil {
			return 0, err
		}
		n, err := svc.TrashIDs(ids)
		if err != nil {
			return 0, err
		}
		fmt.Fprintf(e.Stdout, "deleted_unread=%d\n", n)
		return 0, nil
	}
	if len(params) != 2 || params[0] != "message" {
		return 0, errs.Usage("Use: c3po gmail <preset> delete message <id> | delete unread")
	}
	if err := svc.TrashOne(params[1]); err != nil {
		return 0, err
	}
	fmt.Fprintf(e.Stdout, "deleted message_id=%s\n", params[1])
	return 0, nil
}

func (e Env) deleteUnreadAll() (int, error) {
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
		svc, err := e.service(acct)
		if err != nil {
			return 0, err
		}
		ids, err := svc.ListIDs("is:unread -in:sent")
		if err != nil {
			return 0, err
		}
		n, err := svc.TrashIDs(ids)
		if err != nil {
			return 0, err
		}
		total += n
		fmt.Fprintf(e.Stdout, "preset=%s email=%s deleted_unread=%d\n", preset, acct.Email, n)
	}
	fmt.Fprintf(e.Stdout, "delete unread complete: presets=%d deleted_unread=%d\n", len(app.Presets), total)
	return 0, nil
}

func (e Env) saveAttachments(svc Service, preset string, msg api.Message) (int, error) {
	if len(msg.Attachments) == 0 {
		return 0, nil
	}
	dir := e.attachDir(preset, msg.ID)
	n := 0
	for _, att := range msg.Attachments {
		if att.ID == "" || att.Filename == "" {
			continue
		}
		data, err := svc.Attachment(msg.ID, att.ID)
		if err != nil {
			return n, err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return n, err
		}
		path := filepath.Join(dir, filepath.Base(att.Filename))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return n, err
		}
		fmt.Fprintf(e.Stdout, "attachment: %s\n", filepath.Base(path))
		n++
	}
	return n, nil
}

func (e Env) attachDir(preset, id string) string {
	root := e.Dir
	if root == "" {
		root, _ = os.Getwd()
	}
	return filepath.Join(root, "atch_"+preset+"_"+id)
}
