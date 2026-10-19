package gmail

import (
	"fmt"
	"os"
	"strings"

	"github.com/ryangerardwilson/c3po/gmail/internal/config"
	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
)

func (e Env) send(acct config.Account, params []string) (int, error) {
	req, err := parseSend(params)
	if err != nil {
		return 0, err
	}
	sig, err := readSignature(acct.SignatureFile)
	if err != nil {
		return 0, err
	}
	if req.Editor {
		edited, err := e.editorCompose(acct.Email, sig, true)
		if err != nil || edited.To == "" || edited.Subject == "" || edited.Body == "" {
			return 0, err
		}
		req = mergeEditor(req, edited)
	}
	if req.Draft != "" {
		body, err := readDraft(e.paths().Expand(req.Draft))
		if err != nil {
			return 0, err
		}
		req.Body = body
	}
	req.To = resolveContact(req.To, acct.Contacts)
	cc := resolveAll(csv(req.CC), acct.Contacts)
	bcc := resolveAll(csv(req.BCC), acct.Contacts)
	signed := appendSignature(req.Body, sig)
	raw, err := buildMIME(envelope{
		From: acct.Email, To: req.To, Subject: req.Subject, Body: signed,
		CC: cc, BCC: bcc, Files: expandPaths(e, req.Attach),
	})
	if err != nil {
		return 0, err
	}
	svc, err := e.service(acct)
	if err != nil {
		return 0, err
	}
	id, thread, err := svc.Send(raw, "")
	if err != nil {
		if req.Editor {
			fmt.Fprintf(e.Stdout, "editor_draft_recovery:\nto: %s\nsubject: %s\nbody:\n%s\n", req.To, req.Subject, req.Body)
		}
		return 0, err
	}
	fmt.Fprintf(e.Stdout, "sent message_id=%s thread_id=%s\n", id, thread)
	return 0, nil
}

func (e Env) reply(acct config.Account, params []string) (int, error) {
	req, err := parseReply(params)
	if err != nil {
		return 0, err
	}
	sig, err := readSignature(acct.SignatureFile)
	if err != nil {
		return 0, err
	}
	if req.Editor {
		edited, err := e.editorCompose(acct.Email, sig, false)
		if err != nil || edited.Body == "" {
			return 0, err
		}
		req.Body = edited.Body
		req.CC = joinCSV(edited.CC, req.CC)
		req.BCC = joinCSV(edited.BCC, req.BCC)
		req.Attach = append(edited.Attach, req.Attach...)
	}
	if req.Draft != "" {
		body, err := readDraft(e.paths().Expand(req.Draft))
		if err != nil {
			return 0, err
		}
		req.Body = body
	}
	svc, err := e.service(acct)
	if err != nil {
		return 0, err
	}
	id, thread, err := e.sendReply(svc, acct, req, sig)
	if err != nil {
		e.replyRecovery(req)
		return 0, err
	}
	fmt.Fprintf(e.Stdout, "replied message_id=%s thread_id=%s\n", id, thread)
	return 0, nil
}

func (e Env) sendReply(svc Service, acct config.Account, req compose, sig string) (string, string, error) {
	var anchorID, threadID string
	var headers map[string]string
	if req.Thread {
		msgs, err := svc.Thread(req.Target)
		if err != nil {
			return "", "", err
		}
		if len(msgs) == 0 {
			return "", "", errs.API("Could not determine recipient for reply to thread '%s'", req.Target)
		}
		anchor := msgs[len(msgs)-1]
		anchorID, threadID, headers = anchor.ID, anchor.ThreadID, anchor.Headers
	} else {
		msg, err := svc.Get(req.Target)
		if err != nil {
			return "", "", err
		}
		anchorID, threadID, headers = msg.ID, msg.ThreadID, msg.Headers
		_ = anchorID
	}
	to, cc := replyRecipients(headers, acct.Email, req.All)
	if to == "" {
		label := "message '" + req.Target + "'"
		if req.Thread {
			label = "thread '" + req.Target + "'"
		}
		return "", "", errs.API("Could not determine recipient for reply to %s", label)
	}
	extraCC := resolveAll(csv(req.CC), acct.Contacts)
	extraBCC := resolveAll(csv(req.BCC), acct.Contacts)
	cc = normalizeRecipients(append(cc, extraCC...), acct.Email, []string{to})
	bcc := normalizeRecipients(extraBCC, acct.Email, append([]string{to}, cc...))
	raw, err := buildMIME(envelope{
		From: acct.Email, To: to, Subject: replySubject(headerOf(headers, "subject")),
		Body: appendSignature(req.Body, sig), CC: cc, BCC: bcc,
		Files:     expandPaths(e, req.Attach),
		InReplyTo: headerOf(headers, "message-id"),
		Refs:      joinRefs(headerOf(headers, "references"), headerOf(headers, "message-id")),
	})
	if err != nil {
		return "", "", err
	}
	return svc.Send(raw, threadID)
}

func (e Env) replyRecovery(req compose) {
	if !req.Editor {
		return
	}
	fmt.Fprintln(e.Stdout, "editor_draft_recovery:")
	if req.Thread {
		fmt.Fprintf(e.Stdout, "thread_id: %s\n", req.Target)
	} else {
		fmt.Fprintf(e.Stdout, "message_id: %s\n", req.Target)
		fmt.Fprintln(e.Stdout, "hint: if this id is a thread id, use: reply to thread <thread_id> in editor")
	}
	fmt.Fprintf(e.Stdout, "body:\n%s\n", req.Body)
}

func (e Env) preview(acct config.Account, params []string) (int, error) {
	if len(params) == 0 {
		return 0, errs.Usage("Use: c3po gmail <preset> preview send ... | preview reply ...")
	}
	switch params[0] {
	case "send":
		return e.previewSend(acct, params[1:])
	case "reply":
		return e.previewReply(acct, params[1:])
	default:
		return 0, errs.Usage("Use: c3po gmail <preset> preview send ... | preview reply ...")
	}
}

func (e Env) previewSend(acct config.Account, params []string) (int, error) {
	req, err := parseSend(params)
	if err != nil {
		return 0, err
	}
	if req.Editor {
		fmt.Fprintf(e.Stdout, "preview_send from=%s mode=editor side_effects=none\n", acct.Email)
		return 0, nil
	}
	req.To = resolveContact(req.To, acct.Contacts)
	cc := resolveAll(csv(req.CC), acct.Contacts)
	bcc := resolveAll(csv(req.BCC), acct.Contacts)
	fmt.Fprintf(e.Stdout, "preview_send from=%s to=%s subject=%s cc=%s bcc=%s attachments=%s body_chars=%d signature=automatic side_effects=none\n",
		acct.Email, req.To, pythonRepr(req.Subject), orDash(strings.Join(cc, ",")), orDash(strings.Join(bcc, ",")), pathNames(req.Attach), len(req.Body))
	return 0, nil
}

func (e Env) previewReply(acct config.Account, params []string) (int, error) {
	req, err := parseReply(params)
	if err != nil {
		return 0, err
	}
	cc := resolveAll(csv(req.CC), acct.Contacts)
	bcc := resolveAll(csv(req.BCC), acct.Contacts)
	label := "message_id"
	if req.Thread {
		label = "thread_id"
	}
	mode := "body"
	if req.Editor {
		mode = "editor"
	}
	all := "no"
	if req.All {
		all = "yes"
	}
	fmt.Fprintf(e.Stdout, "preview_reply from=%s %s=%s reply_all=%s mode=%s cc=%s bcc=%s attachments=%s body_chars=%d signature=automatic side_effects=none\n",
		acct.Email, label, req.Target, all, mode, orDash(strings.Join(cc, ",")), orDash(strings.Join(bcc, ",")), pathNames(req.Attach), len(req.Body))
	return 0, nil
}

func readSignature(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", errs.Config("Signature file is empty: %s", path)
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", errs.Config("Signature file is empty: %s", path)
	}
	return s, nil
}

func readDraft(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", errs.Usage("Could not read draft file '%s': %s", path, err.Error())
	}
	if strings.TrimSpace(string(b)) == "" {
		return "", errs.Usage("Draft file is empty: %s", path)
	}
	return string(b), nil
}

func appendSignature(body, sig string) string {
	body = strings.TrimRight(body, " \t\r\n")
	block := "-- \n" + strings.TrimSpace(sig)
	if strings.HasSuffix(body, block) {
		return body
	}
	return body + "\n\n" + block
}

func resolveContact(value string, contacts config.Contacts) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "@") {
		return value
	}
	if email, ok := contacts.Get(strings.ToLower(value)); ok {
		return email
	}
	return value
}

func resolveAll(values []string, contacts config.Contacts) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = resolveContact(v, contacts)
	}
	return out
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func pathNames(paths []string) string {
	if len(paths) == 0 {
		return "-"
	}
	return strings.Join(paths, ",")
}

func pythonRepr(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `\'`) + "'"
}

func joinCSV(a, b string) string {
	parts := append(csv(a), csv(b)...)
	return strings.Join(parts, ",")
}

func mergeEditor(req, edited compose) compose {
	req.To = edited.To
	req.Subject = edited.Subject
	req.Body = edited.Body
	req.CC = edited.CC
	req.BCC = edited.BCC
	req.Attach = edited.Attach
	req.Editor = true
	return req
}

func expandPaths(e Env, paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = e.paths().Expand(p)
	}
	return out
}
