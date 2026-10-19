package app

import (
	"os"
	"path/filepath"
	"time"

	"github.com/ryangerardwilson/c3po/o/internal/fsnav"
	"github.com/ryangerardwilson/c3po/o/internal/tui"
)

func (m model) updateAnimationTick(msg listingAnimationTickMsg) (tui.Model, tui.Cmd) {
	if !m.animationActive || msg.generation != m.animationGeneration {
		return m, nil
	}
	m.animationStep++
	if m.animationStep >= m.animationTargetSteps() {
		m.animationStep = m.animationTargetSteps()
		m.animationActive = false
		return m, nil
	}
	return m, m.animationTick()
}

func (m model) updateStructurePoll(msg structurePollMsg) (tui.Model, tui.Cmd) {
	if msg.generation != m.watchGeneration {
		return m, nil
	}
	if msg.snapshot == m.structureSnapshot {
		return m, m.structurePollTick()
	}
	m.applyStructureSnapshot(msg.snapshot)
	if m.animationActive {
		return m, tui.Batch(m.structurePollTick(), m.animationTick())
	}
	return m, m.structurePollTick()
}

// withAnimation adds the commands a key's changes need: a new cascade
// when the listing was rebuilt, and a new poll when the watch restarted.
func (m model) withAnimation(next model, cmd tui.Cmd, animationGeneration, watchGeneration int) (tui.Model, tui.Cmd) {
	cmds := make([]tui.Cmd, 0, 3)
	cmds = append(cmds, cmd)
	if next.animationGeneration != animationGeneration && next.animationActive {
		cmds = append(cmds, next.animationTick())
	}
	if next.watchGeneration != watchGeneration {
		cmds = append(cmds, next.structurePollTick())
	}
	return next, tui.Batch(cmds...)
}

func (m model) structurePollTick() tui.Cmd {
	generation := m.watchGeneration
	dir := m.currentDir
	filter := m.filter
	showHidden := m.showHidden
	previewDepth := m.previewDepth
	interval := m.watchInterval
	if interval <= 0 {
		interval = structurePollInterval
	}
	return tui.Tick(interval, func(time.Time) tui.Msg {
		return structurePollMsg{
			generation: generation,
			snapshot:   fsnav.SnapshotStructure(dir, filter, showHidden, previewDepth),
		}
	})
}

func (m *model) applyStructureSnapshot(snapshot fsnav.StructureSnapshot) {
	if !snapshot.Exists {
		original := m.currentDir
		m.currentDir = nearestExistingDirectory(original)
		m.filter = ""
		m.selected = 0
		m.reload()
		m.status = "directory removed: " + original
		return
	}
	m.reloadPreservingSelection()
	m.status = "listing updated"
}

func (m model) animationTick() tui.Cmd {
	generation := m.animationGeneration
	return tui.Tick(m.animationPhaseDelay(), func(time.Time) tui.Msg {
		return listingAnimationTickMsg{generation: generation}
	})
}

func (m model) animationPhaseDelay() time.Duration {
	count := len(m.previewRows)
	if count <= 0 {
		return 0
	}
	delay := m.animationDelay
	if maxDelay := listingAnimationDuration / time.Duration(count); delay > maxDelay {
		delay = maxDelay
	}
	return delay / 2
}

func (m model) animationTargetSteps() int {
	start, end := m.visibleWindow()
	return (end - start) * 2
}

func (m model) visibleWindow() (int, int) {
	visible := max(8, m.height-6)
	selectedRow := 0
	for index, row := range m.previewRows {
		if row.IsRoot && row.RootIndex == m.selected {
			selectedRow = index
			break
		}
	}
	start := 0
	if selectedRow >= visible {
		start = selectedRow - visible + 1
	}
	end := min(len(m.previewRows), start+visible)
	return start, end
}

func (m *model) reloadPreservingSelection() {
	selectedPath := ""
	if m.selected >= 0 && m.selected < len(m.entries) {
		selectedPath = m.entries[m.selected].Path
	}
	m.reload()
	m.selectPath(selectedPath)
}

func (m *model) selectPath(path string) {
	if path == "" {
		return
	}
	for index, entry := range m.entries {
		if samePath(entry.Path, path) {
			m.selected = index
			return
		}
	}
}

func (m *model) reload() {
	m.entries = fsnav.List(m.currentDir, m.filter, m.showHidden)
	if m.selected >= len(m.entries) {
		m.selected = max(0, len(m.entries)-1)
	}
	m.pruneMarks()
	m.previewRows = fsnav.ListPreview(m.currentDir, m.filter, m.showHidden, m.previewDepth)
	m.structureSnapshot = fsnav.SnapshotStructure(m.currentDir, m.filter, m.showHidden, m.previewDepth)
	m.watchGeneration++
	m.animationGeneration++
	m.animationStep = 0
	m.animationActive = len(m.previewRows) > 0
}

func (m *model) pruneMarks() {
	visible := make(map[string]bool, len(m.entries))
	for _, entry := range m.entries {
		visible[entry.Path] = true
	}
	for path := range m.marks {
		if !visible[path] {
			delete(m.marks, path)
		}
	}
}

func nearestExistingDirectory(path string) string {
	for {
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			return path
		}
		parent := filepath.Dir(path)
		if parent == path {
			return path
		}
		path = parent
	}
}

func samePath(left, right string) bool {
	leftAbs, _ := filepath.Abs(left)
	rightAbs, _ := filepath.Abs(right)
	return leftAbs == rightAbs
}
