package app

import "fmt"

func (m *model) clearVisual() {
	if !m.visual {
		return
	}
	m.visual = false
	const prefix = "visual "
	if len(m.status) > len(prefix) && m.status[:len(prefix)] == prefix {
		m.status = ""
	}
}

func (m model) visualStatus() string {
	lo, hi := m.visualBounds()
	return fmt.Sprintf("visual %d", hi-lo+1)
}

func (m model) visualBounds() (int, int) {
	lo, hi := m.visualAt, m.selected
	if lo > hi {
		lo, hi = hi, lo
	}
	last := len(m.entries) - 1
	if last < 0 {
		return 0, -1
	}
	if lo < 0 {
		lo = 0
	}
	if hi > last {
		hi = last
	}
	if lo > hi {
		lo = hi
	}
	return lo, hi
}

func (m model) rowInVisual(index int) bool {
	if !m.visual || index < 0 || index >= len(m.entries) {
		return false
	}
	lo, hi := m.visualBounds()
	return index >= lo && index <= hi
}

func (m model) selectionPaths() []string {
	if len(m.entries) == 0 || m.selected < 0 || m.selected >= len(m.entries) {
		return nil
	}
	if !m.visual {
		return []string{m.entries[m.selected].Path}
	}
	lo, hi := m.visualBounds()
	if hi < lo {
		return nil
	}
	paths := make([]string, 0, hi-lo+1)
	for i := lo; i <= hi; i++ {
		paths = append(paths, m.entries[i].Path)
	}
	return paths
}
