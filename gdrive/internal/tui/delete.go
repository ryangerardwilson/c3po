package tui

import (
	"context"

	"github.com/ryangerardwilson/c3po/gdrive/internal/driveapi"
)

// deleteItem hides the file at once and asks Drive to trash it on the
// worker. The folder's next listing replaces the cache; a failure brings
// the file back.
func (m *model) deleteItem() bool {
	rows := m.rows()
	if len(rows) == 0 {
		m.status = "no file"
		return false
	}
	entry := rows[m.clampedCursor()]
	if entry.IsDir() {
		m.status = "dd deletes a file"
		return false
	}
	client, parent := m.client, m.parentID
	if m.trashing == nil {
		m.trashing = map[string]bool{}
	}
	m.trashing[entry.ID] = true
	m.writes++
	m.err = nil
	m.status = "deleting " + entry.Name
	m.cursor = m.clampedCursor()
	m.jobs.Go(func() func() {
		err := trashWithin(client, entry.ID)
		var entries []driveapi.NavEntry
		if err == nil {
			entries, err = listWithin(client, parent)
		}
		return func() {
			delete(m.trashing, entry.ID)
			m.writes--
			if err != nil {
				m.fail(err)
				m.cursor = m.clampedCursor()
				return
			}
			m.relist(parent, entries)
			m.err = nil
			m.status = "deleted " + entry.Name
		}
	})
	return false
}

func trashWithin(client Trasher, fileID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), trashWait)
	defer cancel()
	return client.TrashEntry(ctx, fileID)
}
