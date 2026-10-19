package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/o/internal/fsnav"
	opener "github.com/ryangerardwilson/c3po/o/internal/open"
	"github.com/ryangerardwilson/c3po/o/internal/tui"
)

func (m *model) move(delta int) {
	if len(m.entries) == 0 {
		return
	}
	m.selected = (m.selected + delta + len(m.entries)) % len(m.entries)
}

func (m *model) parent() {
	next := filepath.Dir(m.currentDir)
	if next == m.currentDir {
		return
	}
	m.currentDir = next
	m.filter = ""
	m.selected = 0
	m.reload()
}

func (m *model) enter(dir string) {
	m.currentDir = dir
	m.selected = 0
	m.filter = ""
	m.reload()
}

// accept is enter: in browse mode it hands the listed directory to the
// shell and quits; in pick and save modes it activates the current row.
func (m model) accept() (model, tui.Cmd) {
	if m.cfg.Mode == ModeBrowse {
		m.result.ShellCD = m.currentDir
		return m, tui.Quit
	}
	return m.activate()
}

// activate is l: enter a directory, open a file, or confirm a pick.
func (m model) activate() (model, tui.Cmd) {
	if len(m.entries) == 0 {
		return m, nil
	}
	entry := m.entries[m.selected]
	if m.cfg.Mode == ModePick || m.cfg.Mode == ModeSave {
		return m.pickEntry(entry)
	}
	if entry.IsDir {
		m.enter(entry.Path)
		return m, nil
	}
	return m.openEntry(entry)
}

func (m model) pickEntry(entry fsnav.Entry) (model, tui.Cmd) {
	if m.isSelectable(entry) {
		m.result.Selections = []string{entry.Path}
		if m.cfg.Multi && len(m.marks) > 0 {
			m.result.Selections = markedPaths(m.marks)
		}
		return m, tui.Quit
	}
	if entry.IsDir {
		m.enter(entry.Path)
	}
	return m, nil
}

// openEntry opens a file. Text files run the editor on the terminal and
// the screen returns when it exits; other files go to xdg-open.
func (m model) openEntry(entry fsnav.Entry) (model, tui.Cmd) {
	info, err := os.Stat(entry.Path)
	if err != nil || info.IsDir() {
		m.status = "failed to open " + entry.Name
		return m, nil
	}
	if !fsnav.IsTextLike(entry.Path) {
		_ = opener.OpenDetached(entry.Path)
		m.status = "opened " + entry.Name
		return m, nil
	}
	name := entry.Name
	return m, tui.Exec(opener.EditorCommand(entry.Path), func(err error) tui.Msg {
		return openedMsg{name: name, ok: err == nil}
	})
}

func (m model) isSelectable(entry fsnav.Entry) bool {
	switch m.cfg.Allowed {
	case "dir":
		return entry.IsDir
	case "file":
		return !entry.IsDir && m.extensionAllowed(entry.Path)
	default:
		return true
	}
}

func (m model) extensionAllowed(path string) bool {
	if len(m.cfg.Extensions) == 0 {
		return true
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	for _, allowed := range m.cfg.Extensions {
		if strings.ToLower(allowed) == ext {
			return true
		}
	}
	return false
}

func (m *model) toggleMark() {
	if len(m.entries) == 0 {
		return
	}
	path := m.entries[m.selected].Path
	if m.marks[path] {
		delete(m.marks, path)
		m.status = "unmarked"
		return
	}
	m.marks[path] = true
	m.status = "marked"
	m.move(1)
}

func (m *model) copySelection() {
	paths := markedPaths(m.marks)
	if len(paths) == 0 && len(m.entries) > 0 {
		paths = []string{m.entries[m.selected].Path}
	}
	if len(paths) == 0 {
		return
	}
	cmd := execCommand("wl-copy")
	if cmd == nil {
		m.status = "wl-copy not found"
		return
	}
	cmd.Stdin = strings.NewReader(strings.Join(paths, "\n"))
	if err := cmd.Run(); err != nil {
		m.status = "copy failed"
		return
	}
	m.status = "copied"
}

func markedPaths(marks map[string]bool) []string {
	paths := make([]string, 0, len(marks))
	for path := range marks {
		paths = append(paths, path)
	}
	return paths
}

func execCommand(name string, args ...string) *exec.Cmd {
	if _, err := exec.LookPath(name); err != nil {
		return nil
	}
	return exec.Command(name, args...)
}
