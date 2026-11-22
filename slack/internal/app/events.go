package app

import (
	"fmt"
	"strconv"
)

func (rt *Runtime) dispatchEvents(account Account, preset string, args Args) error {
	switch args.EventsAction {
	case "help":
		fmt.Fprintln(rt.Stdout, `Usage:
  c3po slack <preset> events sync
  c3po slack <preset> events status
  c3po slack <preset> events reset cache`)
		return nil
	case "sync":
		_, err := rt.eventsSyncOnce(account, preset, false)
		return err
	case "status":
		return rt.eventsStatus(account, preset)
	case "reset-cache":
		return rt.eventsResetCache(account, preset)
	default:
		return UsageError{Message: "Use: c3po slack <preset> events sync|status|reset cache"}
	}
}

func (rt *Runtime) eventsSyncOnce(account Account, preset string, quiet bool) (int, error) {
	token, err := resolveListToken(account)
	if err != nil {
		return 0, err
	}
	client := rt.slackClient(token)
	auth, err := client.AuthTest()
	if err != nil {
		return 0, err
	}
	selfUserID := auth.UserID
	if selfUserID == "" {
		return 0, UsageError{Message: "Unable to determine the current Slack user."}
	}
	cachePath := eventCacheDBPath(account, preset)
	rows, err := rt.loadRecentConversations(client, selfUserID, "", 100, false, conversationTypesMember)
	if err != nil {
		return 0, err
	}
	stored := 0
	for _, row := range rows {
		_, _ = rt.Cache.StoreConversation(cachePath, row, row.HistoryLoaded)
	}
	limit := accountInt(account, "events_sync_conversation_limit", eventSyncConversationLimit)
	for index, row := range rows {
		if index >= limit {
			break
		}
		entries, err := rt.loadConversationMessages(client, row, selfUserID, 100, cachePath)
		if err != nil {
			continue
		}
		row.Messages = entries
		row.HistoryLoaded = true
		count, _ := rt.Cache.StoreConversation(cachePath, row, true)
		stored += count
	}
	_ = rt.Cache.RecordSync(cachePath, len(rows), stored)
	if !quiet {
		fmt.Fprintf(rt.Stdout, "events_sync conversations=%d messages=%d cache=%s\n", len(rows), stored, cachePath)
	}
	return stored, nil
}

func (rt *Runtime) eventsStatus(account Account, preset string) error {
	paths := eventCachePaths(account, preset)
	state := map[string]any{
		"cache":          paths.DBFile,
		"exists":         fileExists(paths.DBFile),
		"has_user_token": hasToken(account, "user") || readTokenFile(defaultUserTokenFile) != "",
	}
	if fileExists(paths.DBFile) {
		status, err := rt.Cache.Status(paths.DBFile)
		if err == nil {
			state["conversations"] = status.Conversations
			state["messages"] = status.Messages
			state["last_sync_at"] = status.LastSyncAt
			state["last_sync_conversations"] = strconv.Itoa(status.LastSyncConversations)
			state["last_sync_messages"] = strconv.Itoa(status.LastSyncMessages)
		}
	}
	return rt.printJSON(state)
}

func (rt *Runtime) eventsResetCache(account Account, preset string) error {
	paths := eventCachePaths(account, preset)
	rt.Cache.Reset(paths.DBFile)
	fmt.Fprintf(rt.Stdout, "cache reset: %s\n", paths.DBFile)
	return nil
}
