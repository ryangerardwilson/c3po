package tui

func (m *model) deleteItem() bool {
	if len(m.entries) == 0 {
		m.status = "no file"
		return false
	}
	entry := m.entries[m.clampedCursor()]
	if entry.IsDir() {
		m.status = "dd deletes a file"
		return false
	}
	if err := m.client.TrashEntry(entry.ID); err != nil {
		m.err = err
		m.status = err.Error()
		return false
	}
	cursor := m.cursor
	m.err = nil
	m.load(m.parentID)
	if m.err != nil {
		return false
	}
	m.cursor = cursor
	m.cursor = m.clampedCursor()
	m.status = "deleted " + entry.Name
	return false
}
