package app

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/version"
)

func (rt *Runtime) Run(argv []string) error {
	if done, err := rt.runMeta(argv); done {
		return err
	}
	args, err := parseArgs(argv)
	if err != nil {
		return err
	}
	path := configPath("")
	if done, err := rt.runConfig(path, args); done {
		return err
	}
	cfg, err := loadConfig(path)
	if err != nil {
		return err
	}
	if done, err := rt.runAccount(path, cfg, args); done {
		return err
	}
	preset, account, err := selectAccount(cfg, args.Preset)
	if err != nil {
		return err
	}
	if done, err := rt.runContact(path, cfg, preset, args); done {
		return err
	}
	if args.Command == "" {
		return nil
	}
	contacts := contactsForAccount(cfg, account)
	if done, err := rt.runListed(account, preset, contacts, args); done {
		return err
	}
	if done, err := rt.runInspect(account, preset, contacts, args); done {
		return err
	}
	return rt.runWrite(account, contacts, args)
}

func (rt *Runtime) runMeta(argv []string) (bool, error) {
	if len(argv) == 0 || (len(argv) == 1 && argv[0] == "help") {
		fmt.Fprint(rt.Stdout, helpText)
		return true, nil
	}
	if len(argv) == 1 && argv[0] == "version" {
		fmt.Fprintln(rt.Stdout, version.String())
		return true, nil
	}
	if argv[0] == "help" || argv[0] == "version" {
		return true, UsageError{Message: fmt.Sprintf("Use: c3po slack %s", argv[0])}
	}
	return false, nil
}

func (rt *Runtime) runConfig(path string, args Args) (bool, error) {
	if args.Command == "config" {
		cfg, err := loadConfig(path)
		if err != nil {
			return true, err
		}
		return true, rt.showConfigSummary(path, cfg, args.OutputJSON)
	}
	if args.Command == "config-edit" {
		return true, rt.openConfig(path, configBootstrapText)
	}
	return false, nil
}

func (rt *Runtime) runAccount(path string, cfg Config, args Args) (bool, error) {
	switch args.Command {
	case "accounts-list":
		return true, rt.listAccountPresets(cfg, args.OutputJSON)
	case "setup-check":
		return true, rt.setupCheck(path, cfg, args.OutputJSON)
	case "auth":
		if args.AuthList {
			return true, rt.listAccountPresets(cfg, false)
		}
		if err := rt.configureAccount(path, cfg, args); err != nil {
			return true, err
		}
		return true, nil
	case "mra":
		if args.Preset == "" {
			return true, rt.markAllConfiguredNotificationsAsRead(cfg)
		}
	}
	return false, nil
}

func (rt *Runtime) runContact(path string, cfg Config, preset string, args Args) (bool, error) {
	if args.Command != "ac" {
		return false, nil
	}
	label := strings.TrimSpace(args.Label)
	email := strings.TrimSpace(args.Email)
	if label == "" {
		return true, UsageError{Message: "Label cannot be empty."}
	}
	if !strings.Contains(email, "@") {
		return true, UsageError{Message: "Use: c3po slack <preset> contacts add <label> <email>"}
	}
	if err := saveContact(cfg, preset, label, email); err != nil {
		return true, err
	}
	if err := saveConfig(path, cfg); err != nil {
		return true, err
	}
	fmt.Fprintf(rt.Stdout, "Saved contact '%s' -> %s\n", label, email)
	return true, nil
}

func (rt *Runtime) runListed(account Account, preset string, contacts Contacts, args Args) (bool, error) {
	switch args.Command {
	case "events":
		return true, rt.dispatchEvents(account, preset, args)
	case "list-contacts":
		return true, rt.listContactsReport(contacts, args.OutputJSON)
	case "list-channels":
		return true, rt.withListClient(account, args, func(client SlackClient, _ string) error {
			return rt.listMemberChannelsReport(client, args.OutputJSON)
		})
	case "list-dms":
		return true, rt.withListClient(account, args, func(client SlackClient, _ string) error {
			return rt.listMemberDMsReport(client, args.OutputJSON)
		})
	case "ls":
		return true, rt.runListMessages(account, preset, contacts, args)
	case "unread-count":
		return true, rt.runUnread(account, args)
	case "thread":
		return true, rt.withListClient(account, args, func(client SlackClient, _ string) error {
			return rt.listThreadReplies(client, args)
		})
	case "su":
		return true, rt.runSearchUsers(account, contacts, args)
	default:
		return false, nil
	}
}

func (rt *Runtime) withListClient(account Account, args Args, fn func(SlackClient, string) error) error {
	token, err := resolveListToken(account)
	if err != nil {
		return err
	}
	client := rt.slackClientVerbose(token, args.Verbose)
	auth, err := client.AuthTest()
	if err != nil {
		return err
	}
	return fn(client, str(auth["user_id"]))
}

func (rt *Runtime) runListMessages(account Account, preset string, contacts Contacts, args Args) error {
	return rt.withListClient(account, args, func(client SlackClient, selfUserID string) error {
		if selfUserID == "" {
			return UsageError{Message: "Unable to determine the current Slack user."}
		}
		return rt.listMessages(contacts, client, args, selfUserID, eventCacheDBPath(account, preset))
	})
}

func (rt *Runtime) runUnread(account Account, args Args) error {
	token, err := resolveListToken(account)
	if err != nil {
		return err
	}
	client := rt.slackClient(token)
	if _, err := client.AuthTest(); err != nil {
		return err
	}
	return rt.unreadCountReport(client, args.OutputJSON)
}

func (rt *Runtime) runSearchUsers(account Account, contacts Contacts, args Args) error {
	token, err := resolveToken(account)
	if err != nil {
		return err
	}
	client := rt.slackClient(token)
	if _, err := client.AuthTest(); err != nil {
		return err
	}
	return rt.searchUsersAndContacts(contacts, client, args.Query)
}

func (rt *Runtime) runInspect(account Account, preset string, contacts Contacts, args Args) (bool, error) {
	switch args.Command {
	case "preview-post", "preview-reply", "preview-delete", "preview-edit":
		return true, rt.previewSlackAction(args, contacts)
	case "mra":
		return true, rt.runMarkRead(account, preset)
	case "inspect-message", "inspect-conversation", "o", "df":
		return true, rt.runOpen(account, preset, args)
	default:
		return false, nil
	}
}

func (rt *Runtime) runMarkRead(account Account, preset string) error {
	token, err := resolveMarkReadToken(account)
	if err != nil {
		return err
	}
	client := rt.slackClient(token)
	if _, err := client.AuthTest(); err != nil {
		return err
	}
	result, err := rt.markAllUnreadNotificationsAsRead(client, eventCacheDBPath(account, preset), preset)
	if err != nil {
		return err
	}
	if result.Failed > 0 {
		return fmt.Errorf("mark all read failed for %d conversation(s)", result.Failed)
	}
	return nil
}

func (rt *Runtime) runOpen(account Account, preset string, args Args) error {
	readToken, err := resolveListToken(account)
	if err != nil {
		return err
	}
	readClient := rt.slackClient(readToken)
	auth, err := readClient.AuthTest()
	if err != nil {
		return err
	}
	switch args.Command {
	case "inspect-message", "inspect-conversation":
		return rt.inspectSlack(args, readClient)
	case "o":
		selfUserID := str(auth["user_id"])
		if selfUserID == "" {
			return UsageError{Message: "Unable to determine the current Slack user."}
		}
		return rt.openMessages(args.Recipient, readClient, selfUserID, eventCacheDBPath(account, preset))
	case "df":
		return rt.downloadFile(args.Recipient, args.FileID, args.OutputPath, readClient)
	default:
		return nil
	}
}

func (rt *Runtime) runWrite(account Account, contacts Contacts, args Args) error {
	token, err := resolveToken(account)
	if err != nil {
		return err
	}
	client := rt.slackClient(token)
	if _, err := client.AuthTest(); err != nil {
		return err
	}
	switch args.Command {
	case "sc":
		return rt.clearStaleConversations(client)
	case "post":
		return rt.runPost(account, token, contacts, client, args)
	case "reply":
		return rt.runReply(account, token, client, args)
	case "delete":
		return rt.runDelete(account, token, client, args)
	case "edit":
		return rt.runEdit(account, token, client, args)
	default:
		return UsageError{Message: topLevelUsage()}
	}
}

func (rt *Runtime) runPost(account Account, token string, contacts Contacts, client SlackClient, args Args) error {
	directToken := resolveDirectPostToken(account, token)
	lookupToken, err := resolveLookupToken(account, directToken)
	if err != nil {
		return err
	}
	directClient := rt.slackClient(directToken)
	lookupClient := rt.slackClient(lookupToken)
	target, err := resolvePostTarget(rt, args.Recipient, contacts, client, lookupClient, directClient)
	if err != nil {
		return err
	}
	postClient := rt.writeClient(account, token)
	if target.Kind == "email" || target.Kind == "user" {
		postClient = directClient
	}
	message := strings.TrimSpace(args.Message)
	if len(args.Paths) == 0 && message == "" {
		return UsageError{Message: "Use: c3po slack <preset> send to <target> body <message> [attach <path> ...]"}
	}
	var ts string
	var uploaded []string
	if len(args.Paths) > 0 {
		uploaded, ts, err = completeUploadExternalBatch(postClient, target.ChannelID, "", message, args.Paths)
		if err != nil {
			return err
		}
	} else {
		ts, err = sendPost(postClient, target.ChannelID, args.Message, "")
		if err != nil {
			return err
		}
	}
	return rt.printWriteResult(args.OutputJSON, map[string]any{
		"action":  "posted",
		"target":  args.Recipient,
		"kind":    target.Kind,
		"channel": target.ChannelID,
		"ts":      ts,
		"files":   uploaded,
	}, []string{"posted", "target=" + args.Recipient, "kind=" + target.Kind, "channel=" + target.ChannelID}, ts, uploaded)
}

func (rt *Runtime) runReply(account Account, token string, client SlackClient, args Args) error {
	channelID, messageTS, ok := parseMessageID(args.Recipient)
	if !ok {
		return UsageError{Message: "Use: c3po slack <preset> reply to <message_id> body <message> [attach <path> ...]"}
	}
	writeClient := rt.writeClient(account, token)
	threadTS, err := resolveReplyThreadTS(writeClient, channelID, messageTS)
	if err != nil {
		return err
	}
	ts, err := sendPost(writeClient, channelID, args.Message, threadTS)
	if err != nil {
		return err
	}
	uploaded, err := sendAttachments(writeClient, channelID, threadTS, args.Paths)
	if err != nil {
		return err
	}
	return rt.printWriteResult(args.OutputJSON, map[string]any{
		"action":     "replied",
		"message_id": args.Recipient,
		"channel":    channelID,
		"thread_ts":  threadTS,
		"ts":         ts,
		"files":      uploaded,
	}, []string{"replied", "message_id=" + args.Recipient, "channel=" + channelID, "thread_ts=" + threadTS}, ts, uploaded)
}

func (rt *Runtime) runDelete(account Account, token string, client SlackClient, args Args) error {
	channelID, messageTS, ok := parseMessageID(args.Recipient)
	if !ok {
		return UsageError{Message: "Use: c3po slack <preset> delete message <message_id>"}
	}
	writeClient := rt.writeClient(account, token)
	if err := deleteMessage(writeClient, channelID, messageTS); err != nil {
		return err
	}
	return rt.printWriteResult(args.OutputJSON, map[string]any{
		"action":     "deleted",
		"message_id": args.Recipient,
		"channel":    channelID,
		"ts":         messageTS,
	}, []string{"deleted", "message_id=" + args.Recipient, "channel=" + channelID}, "", nil)
}

func (rt *Runtime) runEdit(account Account, token string, client SlackClient, args Args) error {
	channelID, messageTS, ok := parseMessageID(args.Recipient)
	if !ok {
		return UsageError{Message: "Use: c3po slack <preset> edit message <message_id> body <message>"}
	}
	writeClient := rt.writeClient(account, token)
	if err := editMessage(writeClient, channelID, messageTS, args.Message); err != nil {
		return err
	}
	return rt.printWriteResult(args.OutputJSON, map[string]any{
		"action":     "edited",
		"message_id": args.Recipient,
		"channel":    channelID,
		"ts":         messageTS,
		"body":       args.Message,
	}, []string{"edited", "message_id=" + args.Recipient, "channel=" + channelID}, "", nil)
}

func (rt *Runtime) printWriteResult(asJSON bool, payload map[string]any, base []string, ts string, files []string) error {
	if asJSON {
		if ts != "" {
			payload["ts"] = ts
		}
		if files == nil {
			files = []string{}
		}
		if _, ok := payload["files"]; !ok {
			payload["files"] = files
		}
		return rt.printJSON(payload)
	}
	details := append([]string{}, base...)
	if ts != "" {
		details = append(details, "ts="+ts)
	}
	if len(files) > 0 {
		details = append(details, "files="+strings.Join(files, ","))
	}
	fmt.Fprintln(rt.Stdout, strings.Join(details, " "))
	return nil
}

func (rt *Runtime) writeClient(account Account, fallbackToken string) SlackClient {
	userToken := resolveDirectPostToken(account, fallbackToken)
	if userToken != "" && tokenKind(userToken) == "user" {
		return rt.slackClient(userToken)
	}
	return rt.slackClient(fallbackToken)
}

func (rt *Runtime) slackClientVerbose(token string, verbose bool) SlackClient {
	client := rt.slackClient(token)
	if verbose {
		client.Verbose = rt.Stderr
	}
	return client
}

func (rt *Runtime) showConfigSummary(path string, cfg Config, outputJSON bool) error {
	// Never dump raw tokens. Prefer the same redacted summary as setup check.
	return rt.setupCheck(path, cfg, outputJSON)
}

func (rt *Runtime) openConfig(path string, bootstrap string) error {
	// Fail closed unless a real interactive terminal is available for the editor
	// session. Do not trust FileMode alone; re-check process stdio FDs.
	isTTY := isInteractiveTTY
	if rt.IsTTY != nil {
		isTTY = rt.IsTTY
	}
	if !isTTY() {
		return UsageError{Message: "c3po slack config edit requires an interactive TTY on stdin and stdout. Use `c3po slack config` or `c3po slack setup check` for a redacted summary."}
	}
	// Defense in depth: even if a test hook says TTY, never launch the real
	// editor unless the process stdio FDs are terminals.
	if rt.OpenEditor == nil && !isInteractiveTTY() {
		return UsageError{Message: "c3po slack config edit requires an interactive TTY on stdin and stdout. Use `c3po slack config` or `c3po slack setup check` for a redacted summary."}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(bootstrap), 0o600); err != nil {
			return err
		}
	}
	if rt.OpenEditor != nil {
		return rt.OpenEditor(path, bootstrap)
	}
	editor := firstNonEmpty(getenv("EDITOR"), getenv("VISUAL"))
	if editor == "" {
		return UsageError{Message: "No EDITOR/VISUAL set. Export EDITOR and re-run `c3po slack config edit`, or edit the redacted-safe config path after `c3po slack setup check`."}
	}
	cmd := exec.Command("sh", "-c", editor+" \"$1\"", "slack-editor", path)
	// Attach only to the real process terminals — never to captured rt.Stdout
	// buffers that could leak token-bearing file contents into agent logs.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (rt *Runtime) listAccountPresets(cfg Config, outputJSON bool) error {
	accts := accounts(cfg)
	if outputJSON {
		rows := []map[string]any{}
		for _, preset := range sortedPresetKeys(accts) {
			account := accts[preset]
			rows = append(rows, map[string]any{
				"preset":         preset,
				"name":           firstNonEmpty(str(account["name"]), "-"),
				"has_user_token": hasToken(account, "user"),
				"contacts":       len(contactsForAccount(cfg, account)),
			})
		}
		return rt.printJSON(map[string]any{"accounts": rows})
	}
	if len(accts) == 0 {
		fmt.Fprintln(rt.Stdout, "No account presets configured.")
		return nil
	}
	var rows [][]kv
	for _, preset := range sortedPresetKeys(accts) {
		account := accts[preset]
		rows = append(rows, []kv{
			{"preset", preset},
			{"name", firstNonEmpty(str(account["name"]), "-")},
			{"user", redactToken(directToken(account, "user"))},
			{"contacts", fmt.Sprintf("%d", len(contactsForAccount(cfg, account)))},
		})
	}
	rt.printSections(rows)
	return nil
}

func (rt *Runtime) setupCheck(path string, cfg Config, outputJSON bool) error {
	accts := accounts(cfg)
	state := map[string]any{
		"config":          path,
		"exists":          fileExists(path),
		"accounts_count":  len(accts),
		"has_root_tokens": len(tokenMap(cfg)) > 0,
	}
	var rows []map[string]any
	for _, preset := range sortedPresetKeys(accts) {
		account := accts[preset]
		rows = append(rows, map[string]any{
			"preset":         preset,
			"name":           firstNonEmpty(str(account["name"]), "-"),
			"has_user_token": hasToken(account, "user"),
			"contacts":       len(contactsForAccount(cfg, account)),
		})
	}
	state["accounts"] = rows
	if outputJSON {
		return rt.printJSON(state)
	}
	fmt.Fprintf(rt.Stdout, "config: %s\nexists: %v\naccounts_count: %d\n", path, state["exists"], len(accts))
	for _, row := range rows {
		fmt.Fprintln(rt.Stdout)
		keys := []string{"preset", "name", "has_user_token", "contacts"}
		for _, key := range keys {
			fmt.Fprintf(rt.Stdout, "%s: %v\n", key, row[key])
		}
	}
	return nil
}

func (rt *Runtime) configureAccount(configPath string, cfg Config, args Args) error {
	accts := ensureAccounts(cfg)
	raw, _ := accts[args.AuthPreset].(map[string]any)
	if raw == nil {
		raw = map[string]any{}
		accts[args.AuthPreset] = raw
	}
	account := Account(raw)
	if args.AuthImport {
		if token := readTokenFile(defaultUserTokenFile); token != "" {
			args.AuthUserToken = token
		}
	}
	if args.AuthUserToken == "" && args.AuthName == "" {
		return UsageError{Message: "Use: c3po slack auth <preset> user <user_token> [name <name>]"}
	}
	if args.AuthUserToken != "" && tokenKind(args.AuthUserToken) != "user" {
		return UsageError{Message: "user token must start with xoxp-"}
	}
	tokenObject := asMap(account["token"])
	if tokenObject == nil {
		tokenObject = map[string]any{}
	}
	delete(tokenObject, "bot")
	delete(tokenObject, "app")
	if args.AuthUserToken != "" {
		tokenObject["user"] = args.AuthUserToken
	}
	if len(tokenObject) > 0 {
		account["token"] = tokenObject
	}
	if args.AuthName != "" {
		account["name"] = args.AuthName
	}
	if _, ok := account["contacts"]; !ok {
		account["contacts"] = map[string]any{}
	}
	if err := saveConfig(configPath, cfg); err != nil {
		return err
	}
	fmt.Fprintf(rt.Stdout, "Saved Slack account preset %s\n", args.AuthPreset)
	return nil
}

func (rt *Runtime) searchUsersAndContacts(contacts Contacts, client SlackClient, query string) error {
	queryLower := strings.ToLower(query)
	var rows [][]kv
	for label, target := range contacts {
		if strings.Contains(strings.ToLower(label+" "+target), queryLower) {
			rows = append(rows, []kv{{"type", "contact"}, {"label", label}, {"target", target}})
		}
	}
	data, err := client.Request("users.list", map[string]string{"limit": "200"}, false, http.MethodGet, true)
	if err == nil && data["ok"] == true {
		for _, raw := range asList(data["members"]) {
			user := asMap(raw)
			if boolValue(user["deleted"]) || boolValue(user["is_bot"]) {
				continue
			}
			name := displayUser(user, str(user["id"]))
			email := userEmail(user, "-")
			if strings.Contains(strings.ToLower(name+" "+email+" "+str(user["id"])), queryLower) {
				rows = append(rows, []kv{{"type", "user"}, {"name", name}, {"email", email}, {"id", str(user["id"])}})
			}
		}
	}
	if len(rows) == 0 {
		fmt.Fprintln(rt.Stdout, "No Slack users or contacts matched.")
		return nil
	}
	rt.printSections(rows)
	return nil
}

func (rt *Runtime) previewSlackAction(args Args, contacts Contacts) error {
	if len(args.Paths) > 0 {
		for _, path := range args.Paths {
			if _, err := expandExistingPath(path, "attachment"); err != nil {
				return err
			}
		}
	}
	if args.Command == "preview-delete" {
		channelID, messageTS, _ := parseMessageID(args.Recipient)
		payload := map[string]any{
			"action":     "delete",
			"message_id": args.Recipient,
			"channel":    channelID,
			"ts":         messageTS,
			"command":    fmt.Sprintf("c3po slack %s delete message %s", args.Preset, args.Recipient),
		}
		return rt.printPreview(args.OutputJSON, payload, []kv{
			{"action", "delete"},
			{"message_id", args.Recipient},
			{"channel", channelID},
			{"ts", messageTS},
			{"command", str(payload["command"])},
		})
	}
	if args.Command == "preview-edit" {
		channelID, messageTS, _ := parseMessageID(args.Recipient)
		payload := map[string]any{
			"action":     "edit",
			"message_id": args.Recipient,
			"channel":    channelID,
			"ts":         messageTS,
			"body":       args.Message,
			"command":    fmt.Sprintf("c3po slack %s edit message %s body %q", args.Preset, args.Recipient, args.Message),
		}
		return rt.printPreview(args.OutputJSON, payload, []kv{
			{"action", "edit"},
			{"message_id", args.Recipient},
			{"channel", channelID},
			{"ts", messageTS},
			{"body", args.Message},
			{"command", str(payload["command"])},
		})
	}
	if args.Command == "preview-reply" {
		channelID, messageTS, _ := parseMessageID(args.Recipient)
		cmd := fmt.Sprintf("c3po slack %s reply to %s body %q", args.Preset, args.Recipient, args.Message)
		if len(args.Paths) > 0 {
			cmd += " attach " + strings.Join(args.Paths, " attach ")
		}
		payload := map[string]any{
			"action":      "reply",
			"message_id":  args.Recipient,
			"channel":     channelID,
			"thread_ts":   messageTS,
			"body":        args.Message,
			"attachments": args.Paths,
			"command":     cmd,
		}
		return rt.printPreview(args.OutputJSON, payload, []kv{
			{"action", "reply"},
			{"message_id", args.Recipient},
			{"channel", channelID},
			{"thread_ts", messageTS},
			{"body", args.Message},
			{"attachments", strings.Join(args.Paths, ",")},
			{"command", cmd},
		})
	}
	target := args.Recipient
	targetKind := "raw"
	if mapped := contacts[target]; mapped != "" {
		targetKind = "contact"
		target = mapped
	} else if conversationIDRE.MatchString(target) {
		targetKind = "channel"
	} else if userIDRE.MatchString(target) {
		targetKind = "user"
	} else if strings.Contains(target, "@") {
		targetKind = "email"
	} else if channelNameQuery(target) != "" {
		targetKind = "channel_name"
	}
	cmd := fmt.Sprintf("c3po slack %s send to %s body %q", args.Preset, args.Recipient, args.Message)
	if len(args.Paths) > 0 {
		cmd += " attach " + strings.Join(args.Paths, " attach ")
	}
	payload := map[string]any{
		"action":      "send",
		"target":      args.Recipient,
		"target_kind": targetKind,
		"resolved":    target,
		"body":        args.Message,
		"attachments": args.Paths,
		"command":     cmd,
	}
	return rt.printPreview(args.OutputJSON, payload, []kv{
		{"action", "send"},
		{"target", args.Recipient},
		{"target_kind", targetKind},
		{"resolved", target},
		{"body", args.Message},
		{"attachments", strings.Join(args.Paths, ",")},
		{"command", cmd},
	})
}

func (rt *Runtime) printPreview(asJSON bool, payload map[string]any, fields []kv) error {
	if asJSON {
		if payload["attachments"] == nil {
			payload["attachments"] = []string{}
		}
		return rt.printJSON(payload)
	}
	rt.printSections([][]kv{fields})
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
