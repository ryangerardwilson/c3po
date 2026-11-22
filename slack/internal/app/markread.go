package app

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
)

type MarkResult struct {
	Marked int
	Failed int
}

func notificationUnreadCount(row ConversationRow) int {
	return maxInt(row.Unread, maxInt(intValue(row.Info["unread_count_display"], 0), intValue(row.Info["unread_count"], 0)))
}

func unreadNotificationTotals(client ConversationDirectory) (int, int, error) {
	candidates, err := liveNotificationCandidates(client)
	if err != nil {
		return 0, 0, err
	}
	count := 0
	for _, row := range candidates {
		n := notificationUnreadCount(row)
		if n <= 0 {
			n = 1
		}
		count += n
	}
	return count, len(candidates), nil
}

func (rt *Runtime) unreadCountReport(client ConversationDirectory, outputJSON bool) error {
	count, conversations, err := unreadNotificationTotals(client)
	if err != nil {
		return err
	}
	if outputJSON {
		return rt.printJSON(map[string]any{"count": count, "conversations": conversations})
	}
	rt.printSections([][]kv{{
		{Key: "count", Value: strconv.Itoa(count)},
		{Key: "conversations", Value: strconv.Itoa(conversations)},
	}})
	return nil
}

func notificationLatestTS(row ConversationRow) string {
	if row.LatestTS != "" && row.LatestTS != "0" {
		return row.LatestTS
	}
	if latest := asMap(row.Info["latest"]); len(latest) > 0 {
		if ts := str(latest["ts"]); ts != "" {
			return ts
		}
	}
	return ""
}

func mergeNotification(existing, candidate ConversationRow) ConversationRow {
	if existing.ChannelID == "" {
		return candidate
	}
	if tsFloat(notificationLatestTS(candidate)) > tsFloat(notificationLatestTS(existing)) {
		existing.Info = candidate.Info
		existing.LatestTS = candidate.LatestTS
	}
	if candidate.Unread > existing.Unread {
		existing.Unread = candidate.Unread
	}
	if existing.Surface == "" || existing.Surface == "-" {
		existing.Surface = candidate.Surface
	}
	if existing.Conversation == "" || existing.Conversation == "-" {
		existing.Conversation = candidate.Conversation
	}
	if existing.Name == "" || existing.Name == "-" {
		existing.Name = candidate.Name
	}
	if existing.Members == "" || existing.Members == "-" {
		existing.Members = candidate.Members
	}
	if existing.UserID == "" || existing.UserID == "-" {
		existing.UserID = candidate.UserID
	}
	return existing
}

func apiConversationCandidate(channel map[string]any) (ConversationRow, bool) {
	channelID := str(channel["id"])
	if channelID == "" {
		return ConversationRow{}, false
	}
	unread := maxInt(intValue(channel["unread_count_display"], 0), intValue(channel["unread_count"], 0))
	if unread == 0 && boolValue(channel["has_unreads"]) {
		unread = 1
	}
	if unread <= 0 {
		return ConversationRow{}, false
	}
	return ConversationRow{
		ChannelID:    channelID,
		Surface:      conversationSurface(channel, channelID),
		Conversation: channelName(channel, channelID),
		Name:         firstNonEmpty(str(channel["name"]), str(channel["user"]), channelID),
		Members:      firstNonEmpty(str(channel["num_members"]), "-"),
		UserID:       firstNonEmpty(str(channel["user"]), "-"),
		LatestTS:     extractTS(channel),
		Unread:       unread,
		Info:         channel,
	}, true
}

func liveNotificationCandidates(client ConversationDirectory) ([]ConversationRow, error) {
	channels, err := client.ListConversations("im,mpim,public_channel,private_channel")
	if err != nil {
		return nil, err
	}
	var out []ConversationRow
	for _, channel := range channels {
		if row, ok := apiConversationCandidate(channel); ok {
			out = append(out, row)
		}
	}
	return out, nil
}

func notificationCandidatesForMarkRead(client ConversationDirectory, cachePath string) ([]ConversationRow, error) {
	live, err := liveNotificationCandidates(client)
	if err != nil {
		return nil, err
	}
	sort.Slice(live, func(i, j int) bool {
		return tsFloat(notificationLatestTS(live[i])) > tsFloat(notificationLatestTS(live[j]))
	})
	return live, nil
}

func markReadError(row ConversationRow, code string) string {
	errText := firstNonEmpty(code, "unknown_error")
	if errText == "missing_scope" {
		switch row.Surface {
		case "group_dm":
			return "missing_scope:add mpim:write to user token"
		case "dm":
			return "missing_scope:add im:write to user token"
		case "private_channel":
			return "missing_scope:add groups:write to user token"
		case "channel":
			return "missing_scope:add channels:write to user token"
		}
	}
	return errText
}

func (rt *Runtime) markAllUnreadNotificationsAsRead(client Session, cachePath string, preset string) (MarkResult, error) {
	candidates, err := notificationCandidatesForMarkRead(client, cachePath)
	if err != nil {
		return MarkResult{}, err
	}
	var rows [][]kv
	result := MarkResult{}
	for _, row := range candidates {
		unread := notificationUnreadCount(row)
		if unread <= 0 {
			unread = 1
		}
		latestTS := notificationLatestTS(row)
		action := "marked_read"
		if latestTS == "" {
			result.Failed++
			action = "failed:missing_latest"
		} else {
			var refused *APIError
			if err := client.MarkRead(row.ChannelID, latestTS); errors.As(err, &refused) {
				result.Failed++
				action = "failed:" + markReadError(row, refused.Code)
			} else if err != nil {
				result.Failed++
				action = "failed:" + err.Error()
			} else {
				result.Marked++
				_ = rt.Cache.MarkRead(cachePath, row.ChannelID, latestTS)
			}
		}
		out := []kv{}
		if preset != "" {
			out = append(out, kv{"preset", preset})
		}
		out = append(out,
			kv{"surface", firstNonEmpty(row.Surface, "-")},
			kv{"conversation", firstNonEmpty(row.Conversation, row.Name, "-")},
			kv{"channel_id", firstNonEmpty(row.ChannelID, "-")},
		)
		if row.Members != "" && row.Members != "-" {
			out = append(out, kv{"members", row.Members})
		}
		out = append(out,
			kv{"unread", fmt.Sprintf("%d", unread)},
			kv{"date", formatTS(latestTS)},
			kv{"action", action},
		)
		rows = append(rows, out)
	}
	if len(rows) == 0 {
		suffix := ""
		if preset != "" {
			suffix = " for preset=" + preset
		}
		fmt.Fprintf(rt.Stdout, "No unread notifications to mark as read%s.\n", suffix)
	} else {
		rt.printSections(rows)
		fmt.Fprintf(rt.Stdout, "Summary: marked_read=%d failed=%d\n", result.Marked, result.Failed)
	}
	return result, nil
}

func (rt *Runtime) markAllConfiguredNotificationsAsRead(cfg Config) error {
	accts := accounts(cfg)
	if len(accts) == 0 {
		token, err := resolveMarkReadToken(cfg)
		if err != nil {
			return err
		}
		client := rt.slackClient(token)
		if _, err := client.AuthTest(); err != nil {
			return err
		}
		result, err := rt.markAllUnreadNotificationsAsRead(client, "", "")
		if err != nil {
			return err
		}
		if result.Failed > 0 {
			return fmt.Errorf("mark all read failed for %d conversation(s)", result.Failed)
		}
		return nil
	}
	total := MarkResult{}
	for _, preset := range sortedPresetKeys(accts) {
		account := accts[preset]
		token, err := resolveMarkReadToken(account)
		if err != nil {
			total.Failed++
			fmt.Fprintf(rt.Stdout, "preset=%s failed: %s\n", preset, err)
			continue
		}
		client := rt.slackClient(token)
		if _, err := client.AuthTest(); err != nil {
			total.Failed++
			fmt.Fprintf(rt.Stdout, "preset=%s failed: %s\n", preset, err)
			continue
		}
		result, err := rt.markAllUnreadNotificationsAsRead(client, eventCacheDBPath(account, preset), preset)
		if err != nil {
			total.Failed++
			fmt.Fprintf(rt.Stdout, "preset=%s failed: %s\n", preset, err)
			continue
		}
		total.Marked += result.Marked
		total.Failed += result.Failed
	}
	fmt.Fprintf(rt.Stdout, "All presets summary: marked_read=%d failed=%d\n", total.Marked, total.Failed)
	if total.Failed > 0 {
		return fmt.Errorf("mark all read failed for %d preset/conversation item(s)", total.Failed)
	}
	return nil
}
