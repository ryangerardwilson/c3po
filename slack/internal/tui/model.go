// Package tui is the r2d2 slack screen.
package tui

import (
	"errors"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/tuisession"
)

const (
	styleTitle  = "\x1b[1m"
	styleActive = "\x1b[1;97m"
	styleMuted  = "\x1b[90m"
	styleError  = "\x1b[7m"
	styleReset  = "\x1b[0m"
)

type modeKind int

const (
	modeWorkspaces modeKind = iota
	modeConversations
	modeMessages
)

// Preset is one workspace row.
type Preset struct {
	ID   string
	Name string
}

// Conversation is one channel or direct message.
type Conversation struct {
	ID      string
	Name    string
	Surface string
	Unread  int
	Latest  string
}

// Message is one history row.
type Message struct {
	TS       string
	ThreadTS string
	User     string
	Text     string
}

// Store reads and writes one workspace. Implementations must not print the token.
type Store interface {
	Conversations(preset string) ([]Conversation, error)
	History(preset, channel string) ([]Message, error)
	Send(preset, channel, text, thread string) error
	Mark(preset, channel, ts string) error
}

type screen interface {
	size() (width, height int)
	paint(frame string)
	readKey() (string, error)
}

type model struct {
	presets       []Preset
	preset        int
	conversations []Conversation
	conv          int
	messages      []Message
	cursor        int
	mode          modeKind
	status        string
	err           error
	help          bool
	compose       bool
	draft         string
	thread        string
	width         int
	height        int
	store         Store
	session       *tuisession.Session
}

func newModel(presets []Preset, store Store, sess *tuisession.Session) *model {
	return &model{presets: presets, store: store, session: sess}
}

func (m *model) loop(term screen) error {
	m.restore()
	if m.session != nil {
		stop := startSnapshots()
		defer stop()
	}
	for {
		m.width, m.height = term.size()
		term.paint(m.render())
		key, err := term.readKey()
		if err != nil {
			return err
		}
		if m.handleKey(key) {
			m.noteSession()
			return nil
		}
		m.noteSession()
	}
}

func (m *model) handleKey(key string) bool {
	if m.help {
		return m.helpKey(key)
	}
	if m.compose {
		return m.composeKey(key)
	}
	switch key {
	case "?":
		m.help = true
		m.status = ""
		return false
	case "ctrl+c", "q":
		return true
	case "esc":
		return m.back()
	case "j", "down":
		m.move(1)
	case "k", "up":
		m.move(-1)
	case "g", "home":
		m.cursor = 0
		m.followCursor()
	case "G", "end":
		m.cursor = m.count() - 1
		m.followCursor()
	case "h", "left":
		if m.mode == modeMessages {
			m.back()
			return false
		}
		m.switchPreset(-1)
	case "l", "right":
		if m.mode == modeMessages {
			return false
		}
		m.switchPreset(1)
	case "enter":
		m.open()
	case "i":
		m.startCompose("")
	case "r":
		m.startReply()
	case "m":
		m.mark()
	}
	return false
}

func (m *model) helpKey(key string) bool {
	switch key {
	case "?", "esc":
		m.help = false
	case "ctrl+c", "q":
		return true
	}
	return false
}

func (m *model) composeKey(key string) bool {
	switch key {
	case "esc":
		m.compose = false
		m.draft = ""
		m.thread = ""
		return false
	case "ctrl+c":
		return true
	case "enter":
		m.sendDraft()
		return false
	case "backspace":
		m.backspace()
		return false
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			m.draft += key
		}
		return false
	}
}

func (m *model) backspace() {
	if m.draft == "" {
		return
	}
	m.draft = m.draft[:len(m.draft)-1]
}

func (m *model) move(delta int) {
	m.cursor += delta
	m.followCursor()
}

func (m *model) followCursor() {
	m.cursor = m.clamp()
	if m.mode == modeWorkspaces && len(m.presets) > 0 {
		m.preset = m.cursor
	}
	if m.mode == modeConversations && len(m.conversations) > 0 {
		m.conv = m.cursor
	}
}

func (m *model) back() bool {
	m.compose = false
	m.draft = ""
	m.thread = ""
	switch m.mode {
	case modeMessages:
		m.mode = modeConversations
		m.cursor = m.conv
		return false
	case modeConversations:
		m.mode = modeWorkspaces
		m.cursor = m.preset
		return false
	default:
		return true
	}
}

func (m *model) switchPreset(delta int) {
	if len(m.presets) == 0 || m.mode == modeMessages {
		return
	}
	next := m.preset + delta
	if next < 0 || next >= len(m.presets) {
		return
	}
	m.preset = next
	if m.mode == modeWorkspaces {
		m.cursor = next
		return
	}
	m.cursor = 0
	m.conv = 0
	m.status = ""
	m.err = nil
	m.loadConversations()
}

func (m *model) open() {
	switch m.mode {
	case modeWorkspaces:
		m.openWorkspace()
	case modeConversations:
		m.openConversation()
	}
}

func (m *model) openWorkspace() {
	if len(m.presets) == 0 {
		m.err = nil
		m.status = "no workspaces"
		return
	}
	m.mode = modeConversations
	m.cursor = 0
	m.conv = 0
	m.status = ""
	m.loadConversations()
}

func (m *model) openConversation() {
	if len(m.conversations) == 0 {
		m.err = nil
		m.status = "no conversations"
		return
	}
	m.conv = m.clamp()
	m.mode = modeMessages
	m.status = ""
	m.loadMessages()
	m.cursor = 0
	if len(m.messages) > 0 {
		m.cursor = len(m.messages) - 1
	}
}

func (m *model) startCompose(thread string) {
	if m.mode != modeMessages {
		m.status = "open a conversation"
		m.err = nil
		return
	}
	if _, ok := m.currentConv(); !ok {
		m.status = "open a conversation"
		m.err = nil
		return
	}
	m.compose = true
	m.draft = ""
	m.thread = thread
	m.err = nil
	m.status = ""
}

func (m *model) startReply() {
	msg, ok := m.currentMessage()
	if !ok {
		m.startCompose("")
		return
	}
	thread := msg.TS
	if msg.ThreadTS != "" {
		thread = msg.ThreadTS
	}
	m.startCompose(thread)
}

func (m *model) sendDraft() {
	text := strings.TrimSpace(m.draft)
	if text == "" {
		m.compose = false
		m.draft = ""
		m.thread = ""
		return
	}
	conv, ok := m.currentConv()
	if !ok || m.store == nil {
		m.fail(errors.New("open a conversation"))
		return
	}
	err := m.store.Send(m.presetID(), conv.ID, text, m.thread)
	if err != nil {
		m.fail(err)
		return
	}
	m.compose = false
	m.draft = ""
	m.thread = ""
	m.loadMessages()
	if m.err != nil {
		return
	}
	if len(m.messages) > 0 {
		m.cursor = len(m.messages) - 1
	}
	m.status = "sent"
}

func (m *model) mark() {
	switch m.mode {
	case modeConversations:
		m.markConversation(m.clamp())
	case modeMessages:
		m.markOpen()
	default:
		m.err = nil
		m.status = "open a conversation"
	}
}

func (m *model) markConversation(index int) {
	if index < 0 || index >= len(m.conversations) {
		m.err = nil
		m.status = "nothing to mark"
		return
	}
	row := m.conversations[index]
	if row.Latest == "" {
		m.err = nil
		m.status = "nothing to mark"
		return
	}
	m.markRow(index, row.Latest)
}

func (m *model) markOpen() {
	if len(m.messages) == 0 || m.conv < 0 || m.conv >= len(m.conversations) {
		m.err = nil
		m.status = "nothing to mark"
		return
	}
	m.markRow(m.conv, m.messages[len(m.messages)-1].TS)
}

func (m *model) markRow(index int, ts string) {
	if m.store == nil {
		m.fail(errors.New("slack is unavailable"))
		return
	}
	row := m.conversations[index]
	err := m.store.Mark(m.presetID(), row.ID, ts)
	if err != nil {
		m.fail(err)
		return
	}
	row.Unread = 0
	m.conversations[index] = row
	m.err = nil
	m.status = "marked"
}

func (m *model) fail(err error) {
	m.err = err
	m.status = err.Error()
}

func (m *model) loadConversations() {
	if m.store == nil {
		m.fail(errors.New("slack is unavailable"))
		m.conversations = nil
		return
	}
	rows, err := m.store.Conversations(m.presetID())
	if err != nil {
		m.fail(err)
		m.conversations = nil
		return
	}
	m.conversations = rows
	m.err = nil
}

func (m *model) loadMessages() {
	conv, ok := m.currentConv()
	if !ok || m.store == nil {
		m.messages = nil
		m.fail(errors.New("open a conversation"))
		return
	}
	rows, err := m.store.History(m.presetID(), conv.ID)
	if err != nil {
		m.fail(err)
		m.messages = nil
		return
	}
	m.messages = rows
	m.err = nil
}

func (m *model) currentConv() (Conversation, bool) {
	if m.conv < 0 || m.conv >= len(m.conversations) {
		return Conversation{}, false
	}
	return m.conversations[m.conv], true
}

func (m *model) currentMessage() (Message, bool) {
	index := m.clamp()
	if m.mode != modeMessages || index < 0 || index >= len(m.messages) {
		return Message{}, false
	}
	return m.messages[index], true
}

func (m *model) presetID() string {
	if m.preset < 0 || m.preset >= len(m.presets) {
		return ""
	}
	return m.presets[m.preset].ID
}

func (m *model) count() int {
	switch m.mode {
	case modeWorkspaces:
		return len(m.presets)
	case modeConversations:
		return len(m.conversations)
	default:
		return len(m.messages)
	}
}

func (m *model) clamp() int {
	n := m.count()
	if n == 0 || m.cursor < 0 {
		return 0
	}
	if m.cursor >= n {
		return n - 1
	}
	return m.cursor
}
