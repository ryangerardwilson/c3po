package app

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// The core reaches Slack and the event cache only through the interfaces in
// this file. Slack objects (a conversation, a message, a user) stay the
// loosely typed maps Slack sends; the request and reply envelopes stay in the
// adapter.

// ConnectOptions says how to reach Slack for one user token.
type ConnectOptions struct {
	Token string
	// Verbose receives progress lines (pagination, rate limits). Nil is quiet.
	Verbose io.Writer
	// Timeout bounds each call. Zero keeps the connector's own bound.
	Timeout time.Duration
}

// Connector opens a Gateway for one user token.
type Connector interface {
	Connect(opts ConnectOptions) Gateway
}

// Gateway is everything the core asks of Slack, as the small interfaces below.
type Gateway interface {
	Identity
	ConversationDirectory
	HistoryReader
	UserDirectory
	Messenger
	ConversationMembership
	FileTransfer
}

// Identity names the signed-in user.
type Identity interface {
	AuthTest() (AuthIdentity, error)
}

// AuthIdentity is the signed-in user. UserID is empty when Slack did not say.
type AuthIdentity struct {
	UserID string
}

// ConversationDirectory lists and describes conversations.
type ConversationDirectory interface {
	// ListConversations lists every unarchived conversation of the given
	// comma-separated types that the user is a member of, across all pages.
	ListConversations(types string) ([]map[string]any, error)
	// ListWorkspaceChannels lists every unarchived workspace conversation of
	// the given types, member or not, across all pages.
	ListWorkspaceChannels(types string) ([]map[string]any, error)
	// ConversationInfo describes one conversation. The map may be empty.
	ConversationInfo(query InfoQuery) (map[string]any, error)
}

// InfoQuery names one conversation to describe.
type InfoQuery struct {
	ChannelID string
	// SkipMemberCount asks Slack not to count members.
	SkipMemberCount bool
}

// HistoryReader reads messages.
type HistoryReader interface {
	History(query HistoryQuery) (HistoryPage, error)
	Replies(query RepliesQuery) (HistoryPage, error)
	// SearchMessages returns the matches of a Slack search, newest first.
	// A search Slack refuses returns an *APIError.
	SearchMessages(query string, count int) ([]map[string]any, error)
}

// HistoryQuery selects one page of a conversation's history. Empty fields
// are not sent.
type HistoryQuery struct {
	ChannelID string
	Limit     int
	Oldest    string
	Latest    string
	Inclusive bool
	Cursor    string
}

// RepliesQuery selects one page of a thread.
type RepliesQuery struct {
	ChannelID string
	TS        string
	Limit     int
	Inclusive bool
}

// HistoryPage is one page of messages.
type HistoryPage struct {
	Messages   []map[string]any
	NextCursor string
	// Document is the page as JSON for inspect conversation output json,
	// which prints it unchanged. The core never reads inside it.
	Document json.RawMessage
}

// UserDirectory reads users.
type UserDirectory interface {
	// UserInfo returns one user. The map is empty when Slack sent none.
	UserInfo(userID string) (map[string]any, error)
	// LookupUserByEmail returns one user. The map is empty when Slack sent none.
	LookupUserByEmail(email string) (map[string]any, error)
	// ListUsers returns the first page of the workspace's members.
	ListUsers() ([]map[string]any, error)
}

// Messenger writes messages.
type Messenger interface {
	// OpenDM opens a direct message and returns its conversation, which may be empty.
	OpenDM(userID string) (map[string]any, error)
	// PostMessage posts text and returns its ts. An empty threadTS is a new message.
	PostMessage(channelID, text, threadTS string) (string, error)
	UpdateMessage(channelID, ts, text string) error
	DeleteMessage(channelID, ts string) error
}

// ConversationMembership changes the user's place in a conversation. A call
// Slack refuses returns an *APIError.
type ConversationMembership interface {
	MarkRead(channelID, ts string) error
	CloseConversation(channelID string) error
	LeaveConversation(channelID string) error
}

// FileTransfer moves files to and from Slack.
type FileTransfer interface {
	// UploadFile uploads one file into the conversation and returns its file id.
	UploadFile(channelID, threadTS string, file UploadFile) (string, error)
	// UploadFiles uploads files as one share and returns the share's ts.
	UploadFiles(channelID, threadTS, initialComment string, files []UploadFile) (string, error)
	// DownloadFile saves a private file URL at destination.
	DownloadFile(url, destination string) error
}

// UploadFile is one local file and the name Slack shows for it.
type UploadFile struct {
	Path     string
	Filename string
}

// APIError is a call Slack answered and refused.
type APIError struct {
	Method string
	// Code is Slack's error, or unknown_error.
	Code string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Slack API %s failed: %s", e.Method, e.Code)
}

// Cache is the event cache, as the small interfaces below. Each call names
// the cache file.
type Cache interface {
	CacheWriter
	CacheReader
	CacheLedger
}

// CacheWriter writes conversations and their messages.
type CacheWriter interface {
	// StoreConversation writes a conversation and its messages and returns
	// how many messages it wrote. An empty path writes nothing.
	StoreConversation(path string, row ConversationRow, historyLoaded bool) (int, error)
	// MarkRead moves a cached conversation's last_read to latestTS. An empty
	// path, channel, or ts, or a conversation not cached, does nothing.
	MarkRead(path, channelID, latestTS string) error
}

// CacheReader reads cached messages.
type CacheReader interface {
	// LoadEntries returns up to limit of the newest messages, oldest first,
	// from one conversation or, when channelID is empty, from all. A missing
	// file has none. A message is unread when it is newer than last_read and
	// not selfUserID's.
	LoadEntries(path, selfUserID string, limit int, channelID string) ([]MessageEntry, error)
	// HistoryLoaded says which conversations had their history loaded.
	HistoryLoaded(path string) map[string]bool
}

// CacheLedger keeps the facts about the cache itself.
type CacheLedger interface {
	// RecordSync records one finished sync.
	RecordSync(path string, conversations, messages int) error
	// Status counts the cache and reads the last sync.
	Status(path string) (CacheStatus, error)
	// Reset drops the cache file and its journals.
	Reset(path string)
}

// CacheStatus counts one cache. LastSyncAt is local RFC3339, or empty before
// the first sync.
type CacheStatus struct {
	Conversations         int
	Messages              int
	LastSyncAt            string
	LastSyncConversations int
	LastSyncMessages      int
}
