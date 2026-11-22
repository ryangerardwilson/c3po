package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Session is one Slack user token reached through a Gateway. Verbose
// receives the core's progress lines, the same writer the gateway logs to.
type Session struct {
	Token   string
	Verbose io.Writer
	Gateway
}

func (c Session) logf(format string, args ...any) {
	if c.Verbose == nil {
		return
	}
	fmt.Fprintf(c.Verbose, "slack: "+format+"\n", args...)
}

// session connects one token. verbose sends progress to stderr; a timeout
// above zero bounds each call.
func (rt *Runtime) session(token string, verbose bool, timeout time.Duration) Session {
	opts := ConnectOptions{Token: token, Timeout: timeout}
	if verbose {
		opts.Verbose = rt.Stderr
	}
	return Session{Token: token, Verbose: opts.Verbose, Gateway: rt.Slack.Connect(opts)}
}

type PostTarget struct {
	Kind      string
	ChannelID string
	UserID    string
}

type MessageEntry struct {
	SortTS       float64        `json:"sort_ts"`
	Email        string         `json:"email"`
	ChannelID    string         `json:"channel_id"`
	Surface      string         `json:"surface"`
	Conversation string         `json:"conversation"`
	UserID       string         `json:"user_id"`
	Members      string         `json:"members"`
	Message      map[string]any `json:"message"`
	Sender       map[string]any `json:"sender"`
	Unread       bool           `json:"unread"`
}

type ConversationRow struct {
	ChannelID     string
	Surface       string
	Conversation  string
	Name          string
	Email         string
	Members       string
	UserID        string
	LatestTS      string
	LastRead      string
	Unread        int
	HistoryLoaded bool
	Info          map[string]any
	Messages      []MessageEntry
}

func tokenKind(token string) string {
	switch {
	case strings.HasPrefix(token, "xoxp-"):
		return "user"
	case strings.HasPrefix(token, "xoxb-"):
		return "bot"
	default:
		return "unknown"
	}
}

func requireUserToken(token string, missing string) (string, error) {
	if token == "" {
		return "", UsageError{Message: missing}
	}
	if tokenKind(token) != "user" {
		return "", UsageError{Message: "Slack token must be a user token (xoxp-)."}
	}
	return token, nil
}

func readTokenFile(path string) string {
	path = expandPath(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func tokenMap(cfg map[string]any) map[string]string {
	out := map[string]string{}
	raw := asMap(cfg["token"])
	if value := strings.TrimSpace(str(raw["user"])); value != "" {
		out["user"] = value
	}
	if out["user"] == "" {
		for _, key := range []string{"user_token", "slack_user_token", "token"} {
			if value := strings.TrimSpace(str(cfg[key])); value != "" {
				out["user"] = value
				break
			}
		}
	}
	return out
}

func directToken(cfg map[string]any, keys ...string) string {
	tokens := tokenMap(cfg)
	for _, key := range keys {
		if key != "user" {
			continue
		}
		if value := strings.TrimSpace(tokens[key]); value != "" {
			return value
		}
		if value := strings.TrimSpace(str(cfg[key])); value != "" {
			return value
		}
	}
	return ""
}

func hasToken(cfg map[string]any, kind string) bool {
	if kind != "user" {
		return false
	}
	return directToken(cfg, "user") != ""
}

func resolveUserToken(cfg map[string]any, missing string) (string, error) {
	token := firstNonEmpty(
		directToken(cfg, "user"),
		getenv("SLACK_USER_TOKEN"),
		getenv("SLACK_TOKEN"),
		readTokenFile(defaultUserTokenFile),
	)
	return requireUserToken(token, missing)
}

func resolveToken(cfg map[string]any) (string, error) {
	return resolveUserToken(cfg, "Missing Slack user token (xoxp-). Set SLACK_USER_TOKEN or add token.user in c3po slack config.")
}

func resolveListToken(cfg map[string]any) (string, error) {
	return resolveUserToken(cfg, "Missing Slack user token (xoxp-). For message listing, add token.user in c3po slack config or set SLACK_USER_TOKEN.")
}

func resolveLookupToken(cfg map[string]any, fallback string) (string, error) {
	token := firstNonEmpty(directToken(cfg, "user"), getenv("SLACK_USER_TOKEN"), readTokenFile(defaultUserTokenFile), fallback)
	if token == "" {
		return "", nil
	}
	return requireUserToken(token, "Missing Slack user token (xoxp-).")
}

func resolveDirectPostToken(cfg map[string]any, fallback string) string {
	token := firstNonEmpty(directToken(cfg, "user"), getenv("SLACK_USER_TOKEN"), readTokenFile(defaultUserTokenFile), fallback)
	if tokenKind(token) != "user" {
		return ""
	}
	return token
}

func resolveMarkReadToken(cfg map[string]any) (string, error) {
	return resolveUserToken(cfg, "mark all read requires a user token (xoxp-) with im:write, mpim:write, groups:write, and channels:write for every conversation type.")
}

func (rt *Runtime) slackClient(token string) Session {
	return rt.session(token, false, 0)
}

func channelNameQuery(value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "#") {
		raw = strings.TrimSpace(raw[1:])
	}
	if raw == "" || strings.Contains(raw, "@") || strings.Contains(raw, ":") {
		return ""
	}
	if conversationIDRE.MatchString(raw) || userIDRE.MatchString(raw) {
		return ""
	}
	return strings.ToLower(raw)
}

func listMemberChannels(dir ConversationDirectory) ([]map[string]any, error) {
	return dir.ListConversations("public_channel,private_channel")
}

func listMemberDMs(dir ConversationDirectory) ([]map[string]any, error) {
	return dir.ListConversations("im,mpim")
}

func lookupChannelIDByName(dir ConversationDirectory, value string) (string, error) {
	query := channelNameQuery(value)
	if query == "" {
		return "", nil
	}
	channels, err := listMemberChannels(dir)
	if err != nil {
		return "", err
	}
	var matches []map[string]any
	for _, channel := range channels {
		name := strings.ToLower(firstNonEmpty(str(channel["name"]), str(channel["name_normalized"])))
		if name == query {
			matches = append(matches, channel)
		}
	}
	if len(matches) == 0 {
		return "", nil
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, channel := range matches {
			ids = append(ids, str(channel["id"]))
		}
		sort.Strings(ids)
		return "", UsageError{Message: fmt.Sprintf("Multiple Slack channels named %q. Use an explicit channel id: %s", query, strings.Join(ids, ", "))}
	}
	return str(matches[0]["id"]), nil
}

func lookupUserIDByEmail(users UserDirectory, email string) (string, error) {
	user, err := users.LookupUserByEmail(email)
	if err != nil {
		return "", err
	}
	id := str(user["id"])
	if id == "" {
		return "", fmt.Errorf("No Slack user id found for email: %s", email)
	}
	return id, nil
}

func getUserInfo(users UserDirectory, userID string) (map[string]any, error) {
	user, err := users.UserInfo(userID)
	if err != nil {
		return nil, err
	}
	if len(user) == 0 {
		return nil, fmt.Errorf("No Slack user found for id: %s", userID)
	}
	return user, nil
}

func openDM(messenger Messenger, userID string) (string, error) {
	channel, err := messenger.OpenDM(userID)
	if err != nil {
		return "", err
	}
	channelID := str(channel["id"])
	if channelID == "" {
		return "", fmt.Errorf("Unable to open Slack DM with user: %s", userID)
	}
	return channelID, nil
}

func resolvePostTarget(recipient string, contacts Contacts, lookupClient Session, directClient Session) (PostTarget, error) {
	value := strings.TrimSpace(recipient)
	if mapped := contacts[value]; mapped != "" {
		value = mapped
	}
	if channelID, _, ok := parseMessageID(value); ok {
		return PostTarget{Kind: "message", ChannelID: channelID}, nil
	}
	if conversationIDRE.MatchString(value) {
		return PostTarget{Kind: "channel", ChannelID: value}, nil
	}
	if userIDRE.MatchString(value) {
		channelID, err := openDM(directClient, value)
		if err != nil {
			return PostTarget{}, err
		}
		return PostTarget{Kind: "user", ChannelID: channelID, UserID: value}, nil
	}
	if channelID, err := lookupChannelIDByName(lookupClient, value); err != nil {
		return PostTarget{}, err
	} else if channelID != "" {
		return PostTarget{Kind: "channel_name", ChannelID: channelID}, nil
	}
	if strings.Contains(value, "@") {
		userID, err := lookupUserIDByEmail(lookupClient, value)
		if err != nil {
			return PostTarget{}, err
		}
		channelID, err := openDM(directClient, userID)
		if err != nil {
			return PostTarget{}, err
		}
		return PostTarget{Kind: "email", ChannelID: channelID, UserID: userID}, nil
	}
	return PostTarget{}, UsageError{Message: fmt.Sprintf("Unable to resolve Slack target: %s (use contact, email, #channel, channel id, or message id)", recipient)}
}

func resolveReplyThreadTS(history HistoryReader, channelID, messageTS string) (string, error) {
	page, err := history.Replies(RepliesQuery{ChannelID: channelID, TS: messageTS, Limit: 1, Inclusive: true})
	if err != nil {
		return "", err
	}
	if len(page.Messages) == 0 {
		return "", fmt.Errorf("Message not found: %s:%s", channelID, messageTS)
	}
	message := page.Messages[0]
	threadTS := firstNonEmpty(str(message["thread_ts"]), str(message["ts"]), messageTS)
	return threadTS, nil
}

func expandExistingPath(path string, kind string) (string, error) {
	expanded := expandPath(path)
	info, err := os.Stat(expanded)
	if err != nil {
		return "", fmt.Errorf("%s path not found: %s", kind, path)
	}
	if info.IsDir() && kind != "attachment" {
		return "", fmt.Errorf("%s path is a directory: %s", kind, path)
	}
	return expanded, nil
}

func zipDirectory(dirPath string) (string, func(), error) {
	base := filepath.Base(dirPath)
	temp, err := os.CreateTemp("", base+"-*.zip")
	if err != nil {
		return "", nil, err
	}
	temp.Close()
	cleanup := func() { _ = os.Remove(temp.Name()) }
	err = zipDir(dirPath, temp.Name())
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return temp.Name(), cleanup, nil
}

// uploadJob is one attachment ready to upload. A directory becomes a
// temporary zip that cleanup removes.
type uploadJob struct {
	file    UploadFile
	cleanup func()
}

func prepareUpload(rawPath string) (uploadJob, error) {
	path, err := expandExistingPath(rawPath, "attachment")
	if err != nil {
		return uploadJob{}, err
	}
	cleanup := func() {}
	info, err := os.Stat(path)
	if err != nil {
		return uploadJob{}, err
	}
	filename := filepath.Base(path)
	if info.IsDir() {
		zipped, clean, err := zipDirectory(path)
		if err != nil {
			return uploadJob{}, err
		}
		path = zipped
		cleanup = clean
		filename = filepath.Base(rawPath) + ".zip"
	}
	return uploadJob{file: UploadFile{Path: path, Filename: filename}, cleanup: cleanup}, nil
}

func completeUploadExternalBatch(files FileTransfer, channelID, threadTS, initialComment string, paths []string) ([]string, string, error) {
	if len(paths) == 0 {
		return nil, "", nil
	}
	var jobs []uploadJob
	for _, rawPath := range paths {
		job, err := prepareUpload(rawPath)
		if err != nil {
			for _, job := range jobs {
				job.cleanup()
			}
			return nil, "", err
		}
		jobs = append(jobs, job)
	}
	defer func() {
		for _, job := range jobs {
			job.cleanup()
		}
	}()
	uploads := make([]UploadFile, 0, len(jobs))
	uploaded := make([]string, 0, len(jobs))
	for _, job := range jobs {
		uploads = append(uploads, job.file)
		uploaded = append(uploaded, job.file.Filename)
	}
	shareTS, err := files.UploadFiles(channelID, threadTS, strings.TrimSpace(initialComment), uploads)
	if err != nil {
		return nil, "", err
	}
	return uploaded, shareTS, nil
}

func sendAttachments(files FileTransfer, channelID, threadTS string, paths []string) ([]string, error) {
	var uploaded []string
	for _, rawPath := range paths {
		job, err := prepareUpload(rawPath)
		if err != nil {
			return nil, err
		}
		fileID, err := files.UploadFile(channelID, threadTS, job.file)
		job.cleanup()
		if err != nil {
			return nil, err
		}
		uploaded = append(uploaded, fileID)
	}
	return uploaded, nil
}

func isMessageID(value string) bool {
	_, _, ok := parseMessageID(value)
	return ok
}

func parseMessageID(value string) (string, string, bool) {
	match := messageIDRE.FindStringSubmatch(value)
	if len(match) != 3 {
		return "", "", false
	}
	return match[1], match[2], true
}

func messageID(channelID, ts string) string {
	return channelID + ":" + ts
}

func extractTS(payload map[string]any) string {
	for _, key := range []string{"ts", "latest"} {
		if value := str(payload[key]); value != "" {
			return value
		}
	}
	if latest := asMap(payload["latest"]); len(latest) > 0 {
		return str(latest["ts"])
	}
	return ""
}

func messageText(message map[string]any) string {
	text := strings.TrimSpace(str(message["text"]))
	if text != "" {
		return text
	}
	if files := asList(message["files"]); len(files) > 0 {
		var names []string
		for _, raw := range files {
			file := asMap(raw)
			if name := firstNonEmpty(str(file["name"]), str(file["title"]), str(file["id"])); name != "" {
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			return "files: " + strings.Join(names, ", ")
		}
	}
	return "-"
}

func messageFiles(message map[string]any) []map[string]any {
	var files []map[string]any
	for _, raw := range asList(message["files"]) {
		if file := asMap(raw); len(file) > 0 {
			files = append(files, file)
		}
	}
	return files
}

func summarizeAttachments(message map[string]any) string {
	files := messageFiles(message)
	if len(files) == 0 {
		return "-"
	}
	var names []string
	for _, file := range files {
		names = append(names, firstNonEmpty(str(file["name"]), str(file["title"]), str(file["id"]), "file"))
	}
	return strings.Join(names, ",")
}

func formatTS(value string) string {
	seconds := tsFloat(value)
	if seconds <= 0 {
		return "-"
	}
	return time.Unix(int64(seconds), 0).Local().Format("2006-01-02 15:04")
}

func conversationSurface(info map[string]any, channelID string) string {
	switch {
	case boolValue(info["is_im"]) || strings.HasPrefix(channelID, "D"):
		return "dm"
	case boolValue(info["is_mpim"]) || strings.HasPrefix(channelID, "G"):
		return "group_dm"
	case boolValue(info["is_private"]) || strings.HasPrefix(channelID, "C") && str(info["is_private"]) == "true":
		return "private_channel"
	default:
		return "channel"
	}
}

func channelName(channel map[string]any, channelID string) string {
	return firstNonEmpty(str(channel["name"]), str(channel["user"]), channelID)
}

func displayUser(user map[string]any, fallback string) string {
	profile := asMap(user["profile"])
	return firstNonEmpty(str(profile["display_name"]), str(profile["real_name"]), str(user["real_name"]), str(user["name"]), fallback)
}

func userEmail(user map[string]any, fallback string) string {
	return firstNonEmpty(str(asMap(user["profile"])["email"]), fallback)
}

func senderInfo(users UserDirectory, message map[string]any, userCache map[string]map[string]any) map[string]any {
	userID := str(message["user"])
	if userID == "" {
		userID = str(message["bot_id"])
	}
	if userID == "" {
		return map[string]any{"id": "-", "name": "unknown", "email": "-"}
	}
	if userCache == nil {
		userCache = map[string]map[string]any{}
	}
	user, ok := userCache[userID]
	if !ok {
		loaded, err := getUserInfo(users, userID)
		if err == nil {
			user = loaded
			userCache[userID] = loaded
		}
	}
	return map[string]any{
		"id":    userID,
		"name":  displayUser(user, userID),
		"email": userEmail(user, "-"),
		"label": displayUser(user, userID),
	}
}

func entrySummary(entry MessageEntry) map[string]any {
	ts := str(entry.Message["ts"])
	summary := map[string]any{
		"message_id":      messageID(entry.ChannelID, ts),
		"ts":              ts,
		"surface":         entry.Surface,
		"conversation":    entry.Conversation,
		"sender":          str(entry.Sender["name"]),
		"sender_id":       str(entry.Sender["id"]),
		"sender_email":    str(entry.Sender["email"]),
		"text":            messageText(entry.Message),
		"date":            formatTS(ts),
		"unread":          entry.Unread,
		"attachments":     summarizeAttachments(entry.Message),
		"channel_id":      entry.ChannelID,
		"conversation_id": entry.ChannelID,
	}
	if threadTS := str(entry.Message["thread_ts"]); threadTS != "" {
		summary["thread_ts"] = threadTS
	}
	if replyCount := entry.Message["reply_count"]; replyCount != nil {
		summary["reply_count"] = replyCount
	}
	if latestReply := str(entry.Message["latest_reply"]); latestReply != "" {
		summary["latest_reply"] = latestReply
	}
	return summary
}

func listEntryFields(entry MessageEntry) []kv {
	summary := entrySummary(entry)
	return []kv{
		{"surface", str(summary["surface"])},
		{"conversation", str(summary["conversation"])},
		{"sender", str(summary["sender"])},
		{"date", str(summary["date"])},
		{"unread", str(summary["unread"])},
		{"message_id", str(summary["message_id"])},
		{"text", compactText(str(summary["text"]))},
		{"attachments", str(summary["attachments"])},
	}
}

func inspectEntryFields(entry MessageEntry) []kv {
	summary := entrySummary(entry)
	text := messageText(entry.Message)
	if text == "" || text == "-" {
		text = str(summary["text"])
	}
	return []kv{
		{"surface", str(summary["surface"])},
		{"conversation", str(summary["conversation"])},
		{"sender", str(summary["sender"])},
		{"date", str(summary["date"])},
		{"unread", str(summary["unread"])},
		{"message_id", str(summary["message_id"])},
		{"text", text},
		{"attachments", str(summary["attachments"])},
	}
}

func compactText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "-"
	}
	if len(value) > 240 {
		return value[:237] + "..."
	}
	return value
}

func sortEntriesLatest(entries []MessageEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SortTS > entries[j].SortTS
	})
}
