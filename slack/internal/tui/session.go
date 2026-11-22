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

func (m *model) apply(body tuisession.Place) {
	m.ws = workspaceIndex(m.board.Workspaces, body.Path)
	m.cursor = m.indexOf(body.Cursor)
}

func (m *model) noteSession() {
	if m.session == nil {
		return
	}
	id := ""
	if item, ok := m.current(); ok {
		id = item.MessageID
	}
	_ = m.session.Place(m.presetID(), id)
	_ = m.session.Note(tuisession.Place{})
}

func (m *model) indexOf(id string) int {
	ws, ok := m.workspace()
	if !ok || id == "" {
		return 0
	}
	for i, item := range ws.items() {
		if item.MessageID == id {
			return i
		}
	}
	return 0
}

func workspaceIndex(rows []Workspace, id string) int {
	for i, row := range rows {
		if row.Preset == id {
			return i
		}
	}
	return 0
}
