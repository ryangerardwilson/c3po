package app

import "os"

func (m *model) deleteCurrent() {
	paths := m.selectionPaths()
	if len(paths) == 0 {
		m.visual = false
		m.status = "nothing to delete"
		return
	}
	removed := make([]string, 0, len(paths))
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			m.forgetClip(removed)
			m.reload()
			m.status = err.Error()
			return
		}
		removed = append(removed, path)
	}
	m.forgetClip(removed)
	label := pathNames(paths)
	m.reload()
	m.status = "deleted " + label
}

func (m *model) forgetClip(deleted []string) {
	if !m.clipSet || len(deleted) == 0 {
		return
	}
	kept := make([]string, 0, len(m.clipPaths))
	for _, clip := range m.clipPaths {
		if !clipRemoved(clip, deleted) {
			kept = append(kept, clip)
		}
	}
	if len(kept) == 0 {
		m.clearClip()
		return
	}
	m.clipPaths = kept
}

func clipRemoved(clip string, deleted []string) bool {
	for _, path := range deleted {
		if dirInside(path, clip) {
			return true
		}
	}
	return false
}
