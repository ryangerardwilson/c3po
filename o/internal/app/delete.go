package app

import "os"

func (m *model) deleteCurrent() {
	if len(m.entries) == 0 || m.selected < 0 || m.selected >= len(m.entries) {
		m.status = "nothing to delete"
		return
	}
	entry := m.entries[m.selected]
	if err := os.RemoveAll(entry.Path); err != nil {
		m.status = err.Error()
		return
	}
	if m.clipSet && dirInside(entry.Path, m.clipPath) {
		m.clipSet = false
		m.clipCut = false
		m.clipPath = ""
	}
	m.reload()
	m.status = "deleted " + entry.Name
}
