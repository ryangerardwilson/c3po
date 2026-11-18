package tui

import "github.com/ryangerardwilson/c3po/internal/tuisession"

func (m *model) restore() {
	if m.session == nil {
		return
	}
	body, ok := m.session.Restored()
	if !ok {
		return
	}
	m.apply(body)
}

func (m *model) apply(body tuisession.Body) {
	m.preset = presetIndex(m.presets, body.Path)
	if len(body.Args) > 0 {
		m.mode = parseMode(body.Args[0])
	}
	saved := ""
	if len(body.Args) > 1 {
		saved = body.Args[1]
	}
	if m.mode == modeWorkspaces {
		m.cursor = m.preset
		return
	}
	m.loadConversations()
	if m.mode == modeConversations {
		m.cursor = m.indexOfConv(body.Cursor)
		m.conv = m.cursor
		return
	}
	m.conv = m.indexOfConv(saved)
	m.loadMessages()
	m.cursor = m.indexOfMessage(body.Cursor)
}

func (m *model) noteSession() {
	if m.session == nil {
		return
	}
	conv := ""
	if row, ok := m.currentConv(); ok && m.mode != modeWorkspaces {
		conv = row.ID
	}
	_ = m.session.Place(m.presetID(), m.cursorKey())
	_ = m.session.Note(tuisession.Body{Args: []string{m.modeName(), conv}})
}

func (m *model) cursorKey() string {
	switch m.mode {
	case modeWorkspaces:
		return m.presetID()
	case modeConversations:
		if row, ok := m.currentConv(); ok {
			return row.ID
		}
	default:
		if row, ok := m.currentMessage(); ok {
			return row.TS
		}
	}
	return ""
}

func (m *model) modeName() string {
	switch m.mode {
	case modeConversations:
		return "conversations"
	case modeMessages:
		return "messages"
	default:
		return "workspaces"
	}
}

func parseMode(name string) modeKind {
	switch name {
	case "conversations":
		return modeConversations
	case "messages":
		return modeMessages
	default:
		return modeWorkspaces
	}
}

func presetIndex(presets []Preset, id string) int {
	for i, preset := range presets {
		if preset.ID == id {
			return i
		}
	}
	return 0
}

func (m *model) indexOfConv(id string) int {
	for i, row := range m.conversations {
		if row.ID == id {
			return i
		}
	}
	return 0
}

func (m *model) indexOfMessage(ts string) int {
	for i, row := range m.messages {
		if row.TS == ts {
			return i
		}
	}
	if len(m.messages) == 0 {
		return 0
	}
	return len(m.messages) - 1
}
