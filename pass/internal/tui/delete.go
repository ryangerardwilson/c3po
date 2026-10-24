package tui

func (m *model) deleteItem() bool {
	if len(m.rows) == 0 {
		if m.vaultName == "" {
			m.status = "no vaults"
		} else {
			m.status = "no items"
		}
		return false
	}
	row := m.rows[m.clamp()]
	if row.vault {
		m.status = "dd deletes an item"
		return false
	}
	if err := m.src.Delete(m.vaultName, row.id); err != nil {
		m.fail(err)
		return false
	}
	cursor := m.cursor
	m.loadItems(m.vaultName)
	if m.err != nil {
		return false
	}
	m.cursor = cursor
	m.cursor = m.clamp()
	m.status = "deleted " + row.name
	return false
}
