package app

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (rt *Runtime) listMessages(contacts Contacts, client Session, args Args, selfUserID, cachePath string) error {
	if args.ListLabel != "" {
		if _, ok := contacts[args.ListLabel]; !ok {
			return UsageError{Message: "Unknown contact label: " + args.ListLabel}
		}
	}
	var cutoff float64
	var hasCutoff bool
	if args.ListTimeLimit != "" {
		cutoff, hasCutoff = startTS(args.ListTimeLimit, rt.Now())
		if !hasCutoff {
			return UsageError{Message: "invalid since window " + strconv.Quote(args.ListTimeLimit) + "; use 4h, 2d, 1w, 3m, 1y, YYYY-MM-DD, YYYY-MM, or \"jan 2025\""}
		}
	}
	// Prefer a bounded client for ordinary list calls so agents see failures instead of hangs.
	listClient := Session{
		Token:   client.Token,
		Verbose: client.Verbose,
		Gateway: rt.Slack.Connect(ConnectOptions{Token: client.Token, Verbose: client.Verbose, Timeout: defaultListHTTPTimeout}),
	}
	if hasCutoff {
		listClient.logf("list since=%s cutoff_utc=%s timeout=%s", args.ListTimeLimit, formatCutoffUTC(cutoff), defaultListHTTPTimeout)
	} else {
		listClient.logf("list filter=%s limit=%d timeout=%s", args.ListFilter, args.ListLimit, defaultListHTTPTimeout)
	}
	var entries []MessageEntry
	var err error
	if shouldBypassMessageCache(args) {
		listClient.logf("list source=live reason=filters")
		entries, err = rt.searchMessagesLive(contacts, listClient, args, selfUserID)
	} else {
		listClient.logf("list source=cache path=%s", cachePath)
		entries, err = rt.eventCacheSearchEntries(cachePath, contacts, args.ListLimit, args.ListFilter, selfUserID, args.ListLabel, args.ListFrom, args.ListContains, args.ListTimeLimit, args.ListIn)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			listClient.logf("list cache_miss falling_back=live")
			entries, err = rt.searchMessagesLive(contacts, listClient, args, selfUserID)
		} else {
			listClient.logf("list cache_hit count=%d", len(entries))
		}
	}
	if err != nil {
		return err
	}
	// Re-apply time boundary after merge so limit never truncates older-first.
	if hasCutoff {
		filtered := entries[:0]
		for _, entry := range entries {
			if entry.SortTS >= cutoff {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}
	if len(entries) == 0 {
		if args.OutputJSON {
			payload := map[string]any{"messages": []any{}, "count": 0, "filter": args.ListFilter}
			if hasCutoff {
				payload["since"] = args.ListTimeLimit
				payload["since_cutoff_utc"] = formatCutoffUTC(cutoff)
			}
			if args.ListIn != "" {
				payload["in"] = args.ListIn
			}
			return rt.printJSON(payload)
		}
		msg := fmt.Sprintf("No %s messages found.", args.ListFilter)
		if hasCutoff {
			msg = fmt.Sprintf("No %s messages found since %s (cutoff_utc=%s).", args.ListFilter, args.ListTimeLimit, formatCutoffUTC(cutoff))
		}
		if args.ListIn != "" {
			msg += " in=" + args.ListIn
		}
		fmt.Fprintln(rt.Stdout, msg)
		return nil
	}
	sortEntriesLatest(entries)
	if len(entries) > args.ListLimit {
		entries = entries[:args.ListLimit]
	}
	if args.OpenMode {
		return rt.printOpenEntriesAndMark(entries, client, cachePath)
	}
	if args.OutputJSON {
		summaries := make([]map[string]any, 0, len(entries))
		for _, entry := range entries {
			summaries = append(summaries, entrySummary(entry))
		}
		payload := map[string]any{"messages": summaries, "count": len(summaries), "order": "newest_first"}
		if hasCutoff {
			payload["since"] = args.ListTimeLimit
			payload["since_cutoff_utc"] = formatCutoffUTC(cutoff)
		}
		if args.ListIn != "" {
			payload["in"] = args.ListIn
		}
		return rt.printJSON(payload)
	}
	if hasCutoff {
		fmt.Fprintf(rt.Stdout, "since: %s\nsince_cutoff_utc: %s\norder: newest_first\n\n", args.ListTimeLimit, formatCutoffUTC(cutoff))
	}
	var rows [][]kv
	for _, entry := range entries {
		rows = append(rows, listEntryFields(entry))
	}
	rt.printSections(rows)
	return nil
}

func (rt *Runtime) searchMessagesLive(contacts Contacts, client Session, args Args, selfUserID string) ([]MessageEntry, error) {
	if args.ListFilter == "unread" || args.ListFilter == "read" {
		client.logf("list live=conversations filter=%s", args.ListFilter)
		return rt.scanConversations(contacts, client, args, selfUserID)
	}
	if tokenKind(client.Token) == "user" {
		entries, err := rt.searchMessagesAPI(contacts, client, args, selfUserID)
		if err == nil && entries != nil {
			return entries, nil
		}
	}
	return rt.scanConversations(contacts, client, args, selfUserID)
}

func (rt *Runtime) searchMessagesAPI(contacts Contacts, client Session, args Args, selfUserID string) ([]MessageEntry, error) {
	// Channel-scoped history is more reliable via conversations.history than workspace search.
	if strings.TrimSpace(args.ListIn) != "" {
		client.logf("list live=conversations.history in=%s", args.ListIn)
		return rt.listMessagesInChannel(contacts, client, args, selfUserID)
	}
	query := buildSearchQuery(args, contacts, rt.Now())
	client.logf("list live=search.messages query=%q", query)
	matches, err := client.SearchMessages(query, maxInt(args.ListLimit*4, 20))
	var refused *APIError
	if errors.As(err, &refused) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	cutoff, hasCutoff := startTS(args.ListTimeLimit, rt.Now())
	userCache := map[string]map[string]any{}
	var entries []MessageEntry
	for _, match := range matches {
		channel := asMap(match["channel"])
		channelID := firstNonEmpty(str(channel["id"]), str(match["channel_id"]))
		if channelID == "" {
			continue
		}
		ts := str(match["ts"])
		if ts == "" {
			continue
		}
		message := map[string]any{
			"ts":   ts,
			"text": str(match["text"]),
			"user": str(asMap(match["user"])["id"]),
		}
		if str(message["user"]) == "" {
			message["user"] = str(match["user"])
		}
		sender := senderInfo(client, message, userCache)
		entry := MessageEntry{
			SortTS:       tsFloat(ts),
			Email:        firstNonEmpty(str(sender["email"]), "-"),
			ChannelID:    channelID,
			Surface:      firstNonEmpty(str(channel["type"]), conversationSurface(channel, channelID)),
			Conversation: firstNonEmpty(str(channel["name"]), channelID),
			UserID:       str(sender["id"]),
			Members:      firstNonEmpty(str(channel["num_members"]), "-"),
			Message:      message,
			Sender:       sender,
			Unread:       false,
		}
		if !entryPassesFilters(entry, contacts, args.ListFilter, args.ListFrom, args.ListContains, cutoff, hasCutoff) {
			continue
		}
		if args.ListLabel != "" && !cacheLabelMatches(entry, contacts, args.ListLabel) {
			continue
		}
		if !channelFilterMatches(entry, args.ListIn) {
			continue
		}
		if str(message["user"]) == selfUserID {
			entry.Unread = false
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (rt *Runtime) listMessagesInChannel(contacts Contacts, client Session, args Args, selfUserID string) ([]MessageEntry, error) {
	channelID, label, err := rt.resolveListChannel(client, args.ListIn)
	if err != nil {
		return nil, err
	}
	client.logf("list channel_resolved id=%s label=%s", channelID, label)
	cutoff, hasCutoff := startTS(args.ListTimeLimit, rt.Now())
	query := HistoryQuery{ChannelID: channelID, Limit: maxInt(args.ListLimit*4, 50)}
	if hasCutoff {
		query.Oldest = fmt.Sprintf("%.6f", cutoff)
	}
	client.logf("list conversations.history channel=%s oldest=%s limit=%d", channelID, query.Oldest, query.Limit)
	page, err := client.History(query)
	if err != nil {
		return nil, err
	}
	channelInfo, _ := client.ConversationInfo(InfoQuery{ChannelID: channelID})
	surface := conversationSurface(channelInfo, channelID)
	conversation := firstNonEmpty(channelName(channelInfo, channelID), label, channelID)
	userCache := map[string]map[string]any{}
	var entries []MessageEntry
	for _, message := range page.Messages {
		ts := str(message["ts"])
		if ts == "" {
			continue
		}
		sender := senderInfo(client, message, userCache)
		entry := MessageEntry{
			SortTS:       tsFloat(ts),
			Email:        firstNonEmpty(str(sender["email"]), "-"),
			ChannelID:    channelID,
			Surface:      surface,
			Conversation: conversation,
			UserID:       firstNonEmpty(str(sender["id"]), "-"),
			Members:      firstNonEmpty(str(channelInfo["num_members"]), "-"),
			Message:      message,
			Sender:       sender,
			Unread:       false,
		}
		if !entryPassesFilters(entry, contacts, args.ListFilter, args.ListFrom, args.ListContains, cutoff, hasCutoff) {
			continue
		}
		if args.ListLabel != "" && !cacheLabelMatches(entry, contacts, args.ListLabel) {
			continue
		}
		entries = append(entries, entry)
	}
	sortEntriesLatest(entries)
	return entries, nil
}

func (rt *Runtime) resolveListChannel(client Session, target string) (string, string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", "", UsageError{Message: "in requires: <channel_id|#name|name>"}
	}
	if conversationIDRE.MatchString(target) {
		return target, target, nil
	}
	if channelID, err := lookupChannelIDByName(client, target); err != nil {
		return "", "", err
	} else if channelID != "" {
		return channelID, channelNameQuery(target), nil
	}
	return "", "", UsageError{Message: "Unable to resolve list channel: " + target + " (use channel id like C123 or #name)"}
}

func buildSearchQuery(args Args, contacts Contacts, now time.Time) string {
	var terms []string
	if args.ListContains != "" {
		terms = append(terms, quoteSearch(args.ListContains))
	}
	if args.ListLabel != "" {
		target := contacts[args.ListLabel]
		if strings.Contains(target, "@") {
			terms = append(terms, quoteSearch(target))
		} else if target != "" {
			terms = append(terms, target)
		}
	}
	if args.ListFrom != "" {
		terms = append(terms, "from:"+quoteSearch(resolveSenderSearchTerm(contacts, args.ListFrom)))
	}
	if args.ListIn != "" {
		if conversationIDRE.MatchString(args.ListIn) {
			terms = append(terms, "in:"+args.ListIn)
		} else if name := channelNameQuery(args.ListIn); name != "" {
			terms = append(terms, "in:#"+name)
		} else {
			terms = append(terms, "in:"+quoteSearch(args.ListIn))
		}
	}
	if args.ListTimeLimit != "" {
		if cutoff, ok := startTS(args.ListTimeLimit, now); ok {
			// Slack search after: is date-granular; client still enforces exact cutoff.
			terms = append(terms, "after:"+time.Unix(int64(cutoff), 0).UTC().Add(-24*time.Hour).Format("2006-01-02"))
		}
	}
	if len(terms) == 0 {
		terms = append(terms, "in:im OR in:mpim")
	}
	return strings.Join(terms, " ")
}

func quoteSearch(value string) string {
	value = strings.ReplaceAll(value, `"`, `\"`)
	if strings.ContainsAny(value, " \t") {
		return `"` + value + `"`
	}
	return value
}

func shouldBypassMessageCache(args Args) bool {
	if args.ListFilter == "unread" || args.ListFilter == "read" {
		return true
	}
	return strings.TrimSpace(args.ListFrom) != "" || strings.TrimSpace(args.ListContains) != "" || strings.TrimSpace(args.ListIn) != ""
}

func (rt *Runtime) scanConversations(contacts Contacts, client Session, args Args, selfUserID string) ([]MessageEntry, error) {
	if strings.TrimSpace(args.ListIn) != "" {
		return rt.listMessagesInChannel(contacts, client, args, selfUserID)
	}
	rows, err := rt.loadRecentConversations(client, selfUserID, "", maxInt(args.ListLimit*4, 20), false, conversationTypesMember)
	if err != nil {
		return nil, err
	}
	cutoff, hasCutoff := startTS(args.ListTimeLimit, rt.Now())
	var entries []MessageEntry
	for _, row := range rows {
		if args.ListFilter == "unread" && row.Unread <= 0 {
			continue
		}
		history, err := rt.loadConversationMessages(client, row, selfUserID, maxInt(args.ListLimit*4, 20), "")
		if err != nil {
			continue
		}
		for _, entry := range history {
			if args.ListLabel != "" && !cacheLabelMatches(entry, contacts, args.ListLabel) {
				continue
			}
			if !entryPassesFilters(entry, contacts, args.ListFilter, args.ListFrom, args.ListContains, cutoff, hasCutoff) {
				continue
			}
			entries = append(entries, entry)
		}
	}
	sortEntriesLatest(entries)
	return entries, nil
}

func (rt *Runtime) listThreadReplies(client Session, args Args) error {
	channelID, threadTS, ok := parseMessageID(args.Recipient)
	if !ok {
		return UsageError{Message: "Use: c3po slack <preset> thread <message_id> [limit <count>] [output json]"}
	}
	limit := args.ListLimit
	if limit <= 0 {
		limit = defaultListLimit
	}
	client.logf("thread channel=%s ts=%s limit=%d", channelID, threadTS, limit)
	page, err := client.Replies(RepliesQuery{ChannelID: channelID, TS: threadTS, Limit: maxInt(limit, 1)})
	if err != nil {
		return err
	}
	messages := page.Messages
	if len(messages) == 0 {
		if args.OutputJSON {
			return rt.printJSON(map[string]any{
				"thread_ts":  threadTS,
				"channel_id": channelID,
				"messages":   []any{},
				"count":      0,
			})
		}
		fmt.Fprintln(rt.Stdout, "No thread messages found.")
		return nil
	}
	userCache := map[string]map[string]any{}
	var entries []MessageEntry
	for _, message := range messages {
		ts := str(message["ts"])
		if ts == "" {
			continue
		}
		sender := senderInfo(client, message, userCache)
		entries = append(entries, MessageEntry{
			SortTS:       tsFloat(ts),
			ChannelID:    channelID,
			Surface:      conversationSurface(map[string]any{}, channelID),
			Conversation: channelID,
			Message:      message,
			Sender:       sender,
			UserID:       firstNonEmpty(str(sender["id"]), "-"),
		})
	}
	// Thread chronology is oldest-first (root then replies); agents often need root context first.
	sort.Slice(entries, func(i, j int) bool { return entries[i].SortTS < entries[j].SortTS })
	if len(entries) > limit {
		entries = entries[:limit]
	}
	if args.OutputJSON {
		summaries := make([]map[string]any, 0, len(entries))
		for _, entry := range entries {
			summary := entrySummary(entry)
			summary["thread_ts"] = threadTS
			summary["ts"] = str(entry.Message["ts"])
			summary["reply_count"] = entry.Message["reply_count"]
			summaries = append(summaries, summary)
		}
		nextCursor := page.NextCursor
		return rt.printJSON(map[string]any{
			"thread_ts":   threadTS,
			"channel_id":  channelID,
			"message_id":  args.Recipient,
			"messages":    summaries,
			"count":       len(summaries),
			"order":       "oldest_first",
			"next_cursor": nextCursor,
		})
	}
	fmt.Fprintf(rt.Stdout, "thread: %s\nchannel: %s\ncount: %d\n\n", args.Recipient, channelID, len(entries))
	var rows [][]kv
	for _, entry := range entries {
		fields := inspectEntryFields(entry)
		fields = append(fields, kv{"thread_ts", threadTS})
		rows = append(rows, fields)
	}
	rt.printSections(rows)
	return nil
}

func (rt *Runtime) loadRecentConversations(client Session, selfUserID, cachePath string, limit int, useCache bool, conversationTypes string) ([]ConversationRow, error) {
	if useCache && cachePath != "" {
		rows, err := rt.eventCacheLoadConversationRows(cachePath, selfUserID, limit)
		if err == nil && len(rows) > 0 {
			return rows, nil
		}
	}
	if conversationTypes == "" {
		conversationTypes = conversationTypesDMs
	}
	channels, err := client.ListConversations(conversationTypes)
	if err != nil {
		return nil, err
	}
	userCache := map[string]map[string]any{}
	var rows []ConversationRow
	for _, channel := range channels {
		channelID := str(channel["id"])
		if channelID == "" {
			continue
		}
		info := channel
		surface := conversationSurface(info, channelID)
		label := channelName(info, channelID)
		userID := str(info["user"])
		email := "-"
		if userID != "" && userID != selfUserID {
			user, err := getUserInfo(client, userID)
			if err == nil {
				userCache[userID] = user
				label = displayUser(user, label)
				email = userEmail(user, "-")
			}
		}
		latestTS := extractTS(info)
		unread := maxInt(intValue(info["unread_count_display"], 0), intValue(info["unread_count"], 0))
		if unread == 0 && boolValue(info["has_unreads"]) {
			unread = 1
		}
		rows = append(rows, ConversationRow{
			ChannelID:    channelID,
			Surface:      surface,
			Conversation: label,
			Name:         label,
			Email:        email,
			Members:      firstNonEmpty(str(info["num_members"]), "-"),
			UserID:       firstNonEmpty(userID, "-"),
			LatestTS:     latestTS,
			LastRead:     str(info["last_read"]),
			Unread:       unread,
			Info:         info,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return tsFloat(rows[i].LatestTS) > tsFloat(rows[j].LatestTS) })
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (rt *Runtime) loadConversationMessages(client Session, row ConversationRow, selfUserID string, limit int, cachePath string) ([]MessageEntry, error) {
	page, err := client.History(HistoryQuery{ChannelID: row.ChannelID, Limit: maxInt(1, limit)})
	if err != nil {
		return nil, err
	}
	userCache := map[string]map[string]any{}
	var entries []MessageEntry
	for _, message := range page.Messages {
		ts := str(message["ts"])
		if ts == "" {
			continue
		}
		sender := senderInfo(client, message, userCache)
		unread := row.Unread > 0 && tsFloat(ts) > tsFloat(row.LastRead) && str(message["user"]) != selfUserID
		entries = append(entries, MessageEntry{
			SortTS:       tsFloat(ts),
			Email:        firstNonEmpty(str(sender["email"]), row.Email, "-"),
			ChannelID:    row.ChannelID,
			Surface:      row.Surface,
			Conversation: row.Conversation,
			UserID:       firstNonEmpty(row.UserID, str(sender["id"]), "-"),
			Members:      firstNonEmpty(row.Members, "-"),
			Message:      message,
			Sender:       sender,
			Unread:       unread,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].SortTS < entries[j].SortTS })
	if cachePath != "" {
		row.Messages = entries
		row.HistoryLoaded = true
		_, _ = rt.Cache.StoreConversation(cachePath, row, true)
	}
	return entries, nil
}

func conversationRowsFromEntries(entries []MessageEntry) []ConversationRow {
	byChannel := map[string]ConversationRow{}
	for _, entry := range entries {
		row := byChannel[entry.ChannelID]
		if row.ChannelID == "" {
			row = ConversationRow{
				ChannelID:    entry.ChannelID,
				Surface:      entry.Surface,
				Conversation: entry.Conversation,
				Name:         entry.Conversation,
				Email:        entry.Email,
				Members:      entry.Members,
				UserID:       entry.UserID,
				Info:         map[string]any{"channel_id": entry.ChannelID, "surface": entry.Surface, "conversation": entry.Conversation},
			}
		}
		if entry.SortTS > tsFloat(row.LatestTS) {
			row.LatestTS = str(entry.Message["ts"])
		}
		if entry.Unread {
			row.Unread++
		}
		row.Messages = append(row.Messages, entry)
		byChannel[entry.ChannelID] = row
	}
	rows := make([]ConversationRow, 0, len(byChannel))
	for _, row := range byChannel {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return tsFloat(rows[i].LatestTS) > tsFloat(rows[j].LatestTS) })
	return rows
}

func (rt *Runtime) printOpenEntriesAndMark(entries []MessageEntry, client Session, cachePath string) error {
	latest := map[string]string{}
	for _, entry := range entries {
		rt.printMessageOpen(entry)
		ts := str(entry.Message["ts"])
		if ts == "" {
			continue
		}
		if current := latest[entry.ChannelID]; current == "" || tsFloat(ts) > tsFloat(current) {
			latest[entry.ChannelID] = ts
		}
	}
	marked := 0
	for channelID, ts := range latest {
		if err := client.MarkRead(channelID, ts); err != nil {
			return err
		}
		_ = rt.Cache.MarkRead(cachePath, channelID, ts)
		marked++
	}
	fmt.Fprintf(rt.Stdout, "ls_opened messages=%d marked_conversations=%d\n", len(entries), marked)
	return nil
}

func (rt *Runtime) printMessageOpen(entry MessageEntry) {
	rt.printSections([][]kv{{
		{"surface", entry.Surface},
		{"conversation", entry.Conversation},
		{"sender", firstNonEmpty(str(entry.Sender["name"]), str(entry.Sender["id"]), "-")},
		{"date", formatTS(str(entry.Message["ts"]))},
		{"message_id", messageID(entry.ChannelID, str(entry.Message["ts"]))},
		{"text", messageText(entry.Message)},
		{"attachments", summarizeAttachments(entry.Message)},
	}})
}

func (rt *Runtime) inspectSlack(args Args, client Session) error {
	if args.Command == "inspect-message" {
		channelID, ts, ok := parseMessageID(args.Recipient)
		if !ok {
			return UsageError{Message: "Use: c3po slack <preset> inspect message <message_id>"}
		}
		page, err := client.History(HistoryQuery{ChannelID: channelID, Latest: ts, Inclusive: true, Limit: 1})
		if err != nil {
			return err
		}
		messages := page.Messages
		if len(messages) == 0 {
			return fmt.Errorf("Message not found: %s", args.Recipient)
		}
		message := messages[0]
		if args.OutputJSON {
			return rt.printJSON(map[string]any{"message": message})
		}
		entry := MessageEntry{ChannelID: channelID, Surface: conversationSurface(map[string]any{}, channelID), Conversation: channelID, Message: message, Sender: senderInfo(client, message, nil), SortTS: tsFloat(ts)}
		rt.printSections([][]kv{inspectEntryFields(entry)})
		return nil
	}
	limit := args.ListLimit
	if limit <= 0 {
		limit = defaultListLimit
	}
	page, err := client.History(HistoryQuery{ChannelID: args.Recipient, Limit: limit})
	if err != nil {
		return err
	}
	if args.OutputJSON {
		return rt.printJSON(page.Document)
	}
	var rows [][]kv
	for _, message := range page.Messages {
		entry := MessageEntry{ChannelID: args.Recipient, Surface: conversationSurface(map[string]any{}, args.Recipient), Conversation: args.Recipient, Message: message, Sender: senderInfo(client, message, nil), SortTS: tsFloat(str(message["ts"]))}
		rows = append(rows, inspectEntryFields(entry))
	}
	if len(rows) == 0 {
		fmt.Fprintln(rt.Stdout, "No messages found.")
		return nil
	}
	rt.printSections(rows)
	return nil
}

func (rt *Runtime) openMessages(recipient string, client Session, selfUserID, cachePath string) error {
	if channelID, ts, ok := parseMessageID(recipient); ok {
		args := Args{Command: "inspect-message", Recipient: recipient}
		if err := rt.inspectSlack(args, client); err != nil {
			return err
		}
		if err := client.MarkRead(channelID, ts); err != nil {
			return err
		}
		_ = rt.Cache.MarkRead(cachePath, channelID, ts)
		return nil
	}
	row := ConversationRow{ChannelID: recipient, Surface: conversationSurface(map[string]any{}, recipient), Conversation: recipient}
	entries, err := rt.loadConversationMessages(client, row, selfUserID, defaultListLimit, cachePath)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Fprintln(rt.Stdout, "No messages found.")
		return nil
	}
	if err := rt.printOpenEntriesAndMark(entries, client, cachePath); err != nil {
		return err
	}
	return nil
}

func (rt *Runtime) downloadFile(channelID, fileID, outputPath string, client Session) error {
	cursor := ""
	for {
		history, err := client.History(HistoryQuery{ChannelID: channelID, Limit: 200, Cursor: cursor})
		if err != nil {
			return err
		}
		for _, message := range history.Messages {
			for _, file := range messageFiles(message) {
				if str(file["id"]) != fileID {
					continue
				}
				downloadURL := str(file["url_private_download"])
				if downloadURL == "" {
					return fmt.Errorf("File has no downloadable URL.")
				}
				destination := firstNonEmpty(outputPath, str(file["name"]), fileID)
				destination = expandPath(destination)
				if err := client.DownloadFile(downloadURL, destination); err != nil {
					return err
				}
				fmt.Fprintf(rt.Stdout, "downloaded channel_id=%s file_id=%s path=%s\n", channelID, fileID, destination)
				return nil
			}
		}
		cursor = strings.TrimSpace(history.NextCursor)
		if cursor == "" {
			break
		}
	}
	return fmt.Errorf("File not found in channel_id=%s: %s", channelID, fileID)
}

func (rt *Runtime) clearStaleConversations(client Session) error {
	cutoff := float64(rt.Now().AddDate(0, -6, 0).Unix())
	counts := map[string]int{"closed": 0, "left": 0, "skipped": 0}
	var rows [][]kv
	dms, _ := client.ListConversations("im")
	for _, channel := range dms {
		channelID := str(channel["id"])
		if channelID == "" {
			continue
		}
		info, err := client.ConversationInfo(InfoQuery{ChannelID: channelID, SkipMemberCount: true})
		if err != nil {
			continue
		}
		userID := firstNonEmpty(str(info["user"]), str(channel["user"]), "-")
		user, _ := getUserInfo(client, userID)
		email := userEmail(user, "-")
		latest := extractTS(info)
		var reasons []string
		if email == "-" {
			reasons = append(reasons, "no_email")
		}
		if latest == "" || tsFloat(latest) < cutoff {
			reasons = append(reasons, "stale_6mo")
		}
		if len(reasons) == 0 {
			continue
		}
		action := "closed"
		if err := tolerateRefusal(client.CloseConversation(channelID)); err != nil {
			action = "skip:" + err.Error()
			counts["skipped"]++
		} else {
			counts["closed"]++
		}
		rows = append(rows, []kv{{"type", "dm"}, {"action", action}, {"why", strings.Join(reasons, ",")}, {"name", displayUser(user, userID)}, {"email", email}, {"id", channelID}})
	}
	channels, _ := client.ListWorkspaceChannels("public_channel")
	for _, channel := range channels {
		if !boolValue(channel["is_member"]) {
			continue
		}
		channelID := str(channel["id"])
		if channelID == "" {
			continue
		}
		updated := tsFloat(str(channel["updated"])) / 1000000
		if updated > cutoff {
			continue
		}
		action := "left"
		if boolValue(channel["is_general"]) {
			action = "skip:cant_leave_general"
			counts["skipped"]++
		} else if err := tolerateRefusal(client.LeaveConversation(channelID)); err != nil {
			action = "skip:" + err.Error()
			counts["skipped"]++
		} else {
			counts["left"]++
		}
		rows = append(rows, []kv{{"type", "chan"}, {"action", action}, {"why", "stale_6mo"}, {"name", channelName(channel, channelID)}, {"email", "-"}, {"id", channelID}})
	}
	if len(rows) == 0 {
		fmt.Fprintln(rt.Stdout, "No conversations cleared.")
	} else {
		rt.printSections(rows)
	}
	fmt.Fprintf(rt.Stdout, "Summary: closed=%d left=%d skipped=%d private_and_mpim_skipped=scope\n", counts["closed"], counts["left"], counts["skipped"])
	return nil
}

func (rt *Runtime) listMemberChannelsReport(client Session, outputJSON bool) error {
	channels, err := listMemberChannels(client)
	if err != nil {
		return err
	}
	type row struct {
		Surface   string `json:"surface"`
		Name      string `json:"name"`
		ChannelID string `json:"channel_id"`
		Members   string `json:"members,omitempty"`
	}
	var rows []row
	for _, channel := range channels {
		channelID := str(channel["id"])
		if channelID == "" {
			continue
		}
		surface := conversationSurface(channel, channelID)
		if surface != "channel" && surface != "private_channel" {
			continue
		}
		item := row{
			Surface:   surface,
			Name:      channelName(channel, channelID),
			ChannelID: channelID,
		}
		if members := channel["num_members"]; members != nil {
			item.Members = str(members)
		}
		rows = append(rows, item)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	if outputJSON {
		return rt.printJSON(rows)
	}
	if len(rows) == 0 {
		fmt.Fprintln(rt.Stdout, "No member channels found.")
		return nil
	}
	var sections [][]kv
	for _, item := range rows {
		section := []kv{
			{"surface", item.Surface},
			{"name", item.Name},
			{"channel_id", item.ChannelID},
		}
		if item.Members != "" {
			section = append(section, kv{"members", item.Members})
		}
		sections = append(sections, section)
	}
	rt.printSections(sections)
	return nil
}

func (rt *Runtime) listMemberDMsReport(client Session, outputJSON bool) error {
	channels, err := listMemberDMs(client)
	if err != nil {
		return err
	}
	type row struct {
		Surface   string `json:"surface"`
		Name      string `json:"name"`
		ChannelID string `json:"channel_id"`
	}
	var rows []row
	for _, channel := range channels {
		channelID := str(channel["id"])
		if channelID == "" {
			continue
		}
		surface := conversationSurface(channel, channelID)
		if surface != "dm" && surface != "group_dm" {
			continue
		}
		rows = append(rows, row{
			Surface:   surface,
			Name:      channelName(channel, channelID),
			ChannelID: channelID,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	if outputJSON {
		return rt.printJSON(rows)
	}
	if len(rows) == 0 {
		fmt.Fprintln(rt.Stdout, "No DM or group-DM conversations found.")
		return nil
	}
	var sections [][]kv
	for _, item := range rows {
		sections = append(sections, []kv{
			{"surface", item.Surface},
			{"name", item.Name},
			{"channel_id", item.ChannelID},
		})
	}
	rt.printSections(sections)
	return nil
}

func (rt *Runtime) listContactsReport(contacts Contacts, outputJSON bool) error {
	if len(contacts) == 0 {
		if outputJSON {
			return rt.printJSON([]map[string]string{})
		}
		fmt.Fprintln(rt.Stdout, "No contacts registered.")
		return nil
	}
	keys := make([]string, 0, len(contacts))
	for key := range contacts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if outputJSON {
		payload := make([]map[string]string, 0, len(keys))
		for _, key := range keys {
			payload = append(payload, map[string]string{"label": key, "target": contacts[key]})
		}
		return rt.printJSON(payload)
	}
	var rows [][]kv
	for _, key := range keys {
		rows = append(rows, []kv{{"label", key}, {"target", contacts[key]}})
	}
	rt.printSections(rows)
	return nil
}

// tolerateRefusal drops a refusal Slack answered with and keeps any other
// failure. Conversations clean counts a refused close or leave as done.
func tolerateRefusal(err error) error {
	var refused *APIError
	if errors.As(err, &refused) {
		return nil
	}
	return err
}
