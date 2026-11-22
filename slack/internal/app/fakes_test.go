package app

import (
	"fmt"
	"testing"
)

// fakeSlack is an in-memory Gateway and Connector. A call it was not given
// data for fails the test.
type fakeSlack struct {
	t             *testing.T
	tokens        []string
	userID        string
	conversations map[string][]map[string]any
	users         map[string]map[string]any
	history       map[string][]map[string]any
	markErrs      map[string]error
	marks         []string
}

var _ Connector = (*fakeSlack)(nil)
var _ Gateway = (*fakeSlack)(nil)

func (f *fakeSlack) unexpected(call string) error {
	f.t.Errorf("unexpected Slack call %s", call)
	return fmt.Errorf("unexpected Slack call %s", call)
}

func (f *fakeSlack) Connect(opts ConnectOptions) Gateway {
	f.tokens = append(f.tokens, opts.Token)
	return f
}

func (f *fakeSlack) AuthTest() (AuthIdentity, error) {
	return AuthIdentity{UserID: f.userID}, nil
}

func (f *fakeSlack) ListConversations(types string) ([]map[string]any, error) {
	rows, ok := f.conversations[types]
	if !ok {
		return nil, f.unexpected("ListConversations " + types)
	}
	return rows, nil
}

func (f *fakeSlack) ListWorkspaceChannels(types string) ([]map[string]any, error) {
	return nil, f.unexpected("ListWorkspaceChannels")
}

func (f *fakeSlack) ConversationInfo(query InfoQuery) (map[string]any, error) {
	return nil, f.unexpected("ConversationInfo")
}

func (f *fakeSlack) History(query HistoryQuery) (HistoryPage, error) {
	messages, ok := f.history[query.ChannelID]
	if !ok {
		return HistoryPage{}, f.unexpected("History " + query.ChannelID)
	}
	return HistoryPage{Messages: messages}, nil
}

func (f *fakeSlack) Replies(query RepliesQuery) (HistoryPage, error) {
	return HistoryPage{}, f.unexpected("Replies")
}

func (f *fakeSlack) SearchMessages(query string, count int) ([]map[string]any, error) {
	return nil, f.unexpected("SearchMessages")
}

func (f *fakeSlack) UserInfo(userID string) (map[string]any, error) {
	user, ok := f.users[userID]
	if !ok {
		return nil, f.unexpected("UserInfo " + userID)
	}
	return user, nil
}

func (f *fakeSlack) LookupUserByEmail(email string) (map[string]any, error) {
	return nil, f.unexpected("LookupUserByEmail")
}

func (f *fakeSlack) ListUsers() ([]map[string]any, error) {
	return nil, f.unexpected("ListUsers")
}

func (f *fakeSlack) OpenDM(userID string) (map[string]any, error) {
	return nil, f.unexpected("OpenDM")
}

func (f *fakeSlack) PostMessage(channelID, text, threadTS string) (string, error) {
	return "", f.unexpected("PostMessage")
}

func (f *fakeSlack) UpdateMessage(channelID, ts, text string) error {
	return f.unexpected("UpdateMessage")
}

func (f *fakeSlack) DeleteMessage(channelID, ts string) error {
	return f.unexpected("DeleteMessage")
}

func (f *fakeSlack) MarkRead(channelID, ts string) error {
	f.marks = append(f.marks, channelID+":"+ts)
	return f.markErrs[channelID]
}

func (f *fakeSlack) CloseConversation(channelID string) error {
	return f.unexpected("CloseConversation")
}

func (f *fakeSlack) LeaveConversation(channelID string) error {
	return f.unexpected("LeaveConversation")
}

func (f *fakeSlack) UploadFile(channelID, threadTS string, file UploadFile) (string, error) {
	return "", f.unexpected("UploadFile")
}

func (f *fakeSlack) UploadFiles(channelID, threadTS, initialComment string, files []UploadFile) (string, error) {
	return "", f.unexpected("UploadFiles")
}

func (f *fakeSlack) DownloadFile(url, destination string) error {
	return f.unexpected("DownloadFile")
}

// fakeCache is an in-memory Cache that records the marks it was given.
type fakeCache struct {
	marks []string
}

var _ Cache = (*fakeCache)(nil)

func (c *fakeCache) StoreConversation(path string, row ConversationRow, historyLoaded bool) (int, error) {
	return 0, nil
}

func (c *fakeCache) MarkRead(path, channelID, latestTS string) error {
	c.marks = append(c.marks, path+"|"+channelID+":"+latestTS)
	return nil
}

func (c *fakeCache) LoadEntries(path, selfUserID string, limit int, channelID string) ([]MessageEntry, error) {
	return nil, nil
}

func (c *fakeCache) HistoryLoaded(path string) map[string]bool {
	return map[string]bool{}
}

func (c *fakeCache) RecordSync(path string, conversations, messages int) error {
	return nil
}

func (c *fakeCache) Status(path string) (CacheStatus, error) {
	return CacheStatus{}, nil
}

func (c *fakeCache) Reset(path string) {}
