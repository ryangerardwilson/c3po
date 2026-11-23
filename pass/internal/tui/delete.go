package tui

import "context"

// deleteItem hides the item at once and asks nebula crypt to delete it on
// the worker. The vault's next item list replaces the cache; a failure
// brings the item back.
func (m *model) deleteItem() bool {
	rows := m.shown()
	if len(rows) == 0 {
		if m.place() == "" {
			m.status = "no vaults"
		} else {
			m.status = "no items"
		}
		return false
	}
	row := rows[m.clamp()]
	if row.vault {
		m.status = "dd deletes an item"
		return false
	}
	src, vault := m.src, m.vaultName
	if m.deleting == nil {
		m.deleting = map[string]bool{}
	}
	m.deleting[row.id] = true
	m.writes++
	m.err = nil
	m.status = "deleting " + row.name
	m.cursor = m.clamp()
	m.jobs.Go(func() func() {
		ctx, cancel := context.WithTimeout(context.Background(), cryptWait)
		defer cancel()
		err := src.Delete(ctx, vault, row.id)
		var items []Item
		if err == nil {
			items, err = src.Items(ctx, vault)
		}
		return func() {
			delete(m.deleting, row.id)
			m.writes--
			if err != nil {
				m.fail(err)
				m.cursor = m.clamp()
				return
			}
			if m.vaultName == vault {
				m.rows = itemRows(items)
				m.cursor = m.clamp()
			}
			m.err = nil
			m.status = "deleted " + row.name
		}
	})
	return false
}
