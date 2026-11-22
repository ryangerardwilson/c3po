package slackapi

import (
	"encoding/json"
	"strconv"

	"github.com/ryangerardwilson/c3po/slack/internal/anyval"
	"github.com/ryangerardwilson/c3po/slack/internal/app"
)

// AuthTest names the token's user.
func (c Client) AuthTest() (app.AuthIdentity, error) {
	data, err := c.postJSON("auth.test", map[string]string{})
	if err != nil {
		return app.AuthIdentity{}, err
	}
	return app.AuthIdentity{UserID: anyval.Str(data["user_id"])}, nil
}

// ListConversations pages users.conversations.
func (c Client) ListConversations(types string) ([]map[string]any, error) {
	return listAPI(c, "users.conversations", map[string]string{
		"types":            types,
		"exclude_archived": "true",
		"limit":            "200",
	}, "channels")
}

// ListWorkspaceChannels pages conversations.list.
func (c Client) ListWorkspaceChannels(types string) ([]map[string]any, error) {
	return listAPI(c, "conversations.list", map[string]string{
		"types":            types,
		"exclude_archived": "true",
		"limit":            "200",
	}, "channels")
}

// ConversationInfo reads conversations.info. A refused call still returns
// whatever channel the reply carried.
func (c Client) ConversationInfo(query app.InfoQuery) (map[string]any, error) {
	payload := map[string]string{"channel": query.ChannelID}
	if query.SkipMemberCount {
		payload["include_num_members"] = "false"
	}
	data, err := c.get("conversations.info", payload)
	if data == nil {
		return nil, err
	}
	return anyval.Map(data["channel"]), err
}

// historyPage translates a history reply. Document keeps the whole reply
// as JSON; a decoded reply always marshals.
func historyPage(data map[string]any) app.HistoryPage {
	document, _ := json.Marshal(data)
	return app.HistoryPage{
		Messages:   objects(data["messages"]),
		NextCursor: anyval.Str(anyval.Map(data["response_metadata"])["next_cursor"]),
		Document:   document,
	}
}

// History reads one page of conversations.history.
func (c Client) History(query app.HistoryQuery) (app.HistoryPage, error) {
	payload := map[string]string{"channel": query.ChannelID}
	if query.Limit > 0 {
		payload["limit"] = strconv.Itoa(query.Limit)
	}
	if query.Oldest != "" {
		payload["oldest"] = query.Oldest
	}
	if query.Latest != "" {
		payload["latest"] = query.Latest
	}
	if query.Inclusive {
		payload["inclusive"] = "true"
	}
	if query.Cursor != "" {
		payload["cursor"] = query.Cursor
	}
	data, err := c.get("conversations.history", payload)
	if err != nil {
		return app.HistoryPage{}, err
	}
	return historyPage(data), nil
}

// Replies reads one page of conversations.replies.
func (c Client) Replies(query app.RepliesQuery) (app.HistoryPage, error) {
	payload := map[string]string{
		"channel": query.ChannelID,
		"ts":      query.TS,
		"limit":   strconv.Itoa(query.Limit),
	}
	if query.Inclusive {
		payload["inclusive"] = "true"
	}
	data, err := c.get("conversations.replies", payload)
	if err != nil {
		return app.HistoryPage{}, err
	}
	return historyPage(data), nil
}

// SearchMessages runs search.messages, newest first.
func (c Client) SearchMessages(query string, count int) ([]map[string]any, error) {
	data, err := c.get("search.messages", map[string]string{
		"query":    query,
		"count":    strconv.Itoa(count),
		"sort":     "timestamp",
		"sort_dir": "desc",
	})
	if err != nil {
		return nil, err
	}
	return objects(anyval.Map(data["messages"])["matches"]), nil
}

// UserInfo reads users.info.
func (c Client) UserInfo(userID string) (map[string]any, error) {
	data, err := c.get("users.info", map[string]string{"user": userID})
	if err != nil {
		return nil, err
	}
	return anyval.Map(data["user"]), nil
}

// LookupUserByEmail reads users.lookupByEmail.
func (c Client) LookupUserByEmail(email string) (map[string]any, error) {
	data, err := c.get("users.lookupByEmail", map[string]string{"email": email})
	if err != nil {
		return nil, err
	}
	return anyval.Map(data["user"]), nil
}

// ListUsers reads the first page of users.list.
func (c Client) ListUsers() ([]map[string]any, error) {
	data, err := c.get("users.list", map[string]string{"limit": "200"})
	if err != nil {
		return nil, err
	}
	return objects(data["members"]), nil
}

// OpenDM calls conversations.open.
func (c Client) OpenDM(userID string) (map[string]any, error) {
	data, err := c.postJSON("conversations.open", map[string]string{"users": userID})
	if err != nil {
		return nil, err
	}
	return anyval.Map(data["channel"]), nil
}

// PostMessage calls chat.postMessage.
func (c Client) PostMessage(channelID, text, threadTS string) (string, error) {
	payload := map[string]string{
		"channel": channelID,
		"text":    text,
	}
	if threadTS != "" {
		payload["thread_ts"] = threadTS
	}
	data, err := c.postJSON("chat.postMessage", payload)
	if err != nil {
		return "", err
	}
	return anyval.Str(data["ts"]), nil
}

// UpdateMessage calls chat.update.
func (c Client) UpdateMessage(channelID, ts, text string) error {
	_, err := c.postJSON("chat.update", map[string]string{
		"channel": channelID,
		"ts":      ts,
		"text":    text,
	})
	return err
}

// DeleteMessage calls chat.delete.
func (c Client) DeleteMessage(channelID, ts string) error {
	_, err := c.postJSON("chat.delete", map[string]string{
		"channel": channelID,
		"ts":      ts,
	})
	return err
}

// MarkRead calls conversations.mark.
func (c Client) MarkRead(channelID, ts string) error {
	_, err := c.postForm("conversations.mark", map[string]string{"channel": channelID, "ts": ts})
	return err
}

// CloseConversation calls conversations.close.
func (c Client) CloseConversation(channelID string) error {
	_, err := c.postForm("conversations.close", map[string]string{"channel": channelID})
	return err
}

// LeaveConversation calls conversations.leave.
func (c Client) LeaveConversation(channelID string) error {
	_, err := c.postForm("conversations.leave", map[string]string{"channel": channelID})
	return err
}
