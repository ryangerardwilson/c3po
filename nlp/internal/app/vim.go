package app

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ryangerardwilson/c3po/nlp/internal/course"
)

type scriptEdit struct {
	kind   string
	name   string
	index  int
	path   string
	before string
}

type scriptSavedMsg struct {
	edit scriptEdit
	err  error
}

func defaultFindVim() (string, error) {
	bin, err := exec.LookPath("vim")
	if err != nil {
		return "", errors.New("vim is not on PATH")
	}
	return bin, nil
}

func defaultStartVim(path string, done func(error) tea.Msg) tea.Cmd {
	bin, err := findVim()
	if err != nil {
		return func() tea.Msg { return done(err) }
	}
	return tea.ExecProcess(exec.Command(bin, path), done)
}

var (
	findVim  = defaultFindVim
	startVim = defaultStartVim
)

func (m *model) createCourse(name string) tea.Cmd {
	if m.courseNameTaken(name) {
		m.status = "course already exists"
		return nil
	}
	return m.openPractice("", scriptEdit{kind: "new", name: name})
}

func (m *model) editSelected() tea.Cmd {
	if m.selected < 0 || m.selected >= len(m.pages) {
		m.status = "nothing to edit"
		return nil
	}
	p := m.pages[m.selected]
	item, ok := m.courseAt(p.courseIndex)
	if !ok {
		return nil
	}
	text, err := course.LessonText(item.Body, p.partIndex, p.sectionIndex, p.lessonIndex)
	if err != nil {
		m.status = err.Error()
		return nil
	}
	return m.openPractice(text, scriptEdit{kind: "edit", index: m.selected, before: text})
}

func (m *model) openPractice(text string, edit scriptEdit) tea.Cmd {
	if _, err := findVim(); err != nil {
		m.status = err.Error()
		return nil
	}
	file, err := os.CreateTemp("", "nlp-session-*.txt")
	if err != nil {
		m.status = err.Error()
		return nil
	}
	path := file.Name()
	payload := []byte(text)
	if text != "" && !strings.HasSuffix(text, "\n") {
		payload = append(payload, '\n')
	}
	if _, err := file.Write(payload); err != nil {
		file.Close()
		os.Remove(path)
		m.status = err.Error()
		return nil
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		m.status = err.Error()
		return nil
	}
	edit.path = path
	m.status = "vim"
	return startVim(path, func(err error) tea.Msg {
		return scriptSavedMsg{edit: edit, err: err}
	})
}

func (m *model) finishScript(msg scriptSavedMsg) {
	if msg.edit.path != "" {
		defer os.Remove(msg.edit.path)
	}
	if msg.err != nil {
		m.status = msg.err.Error()
		return
	}
	raw, err := os.ReadFile(msg.edit.path)
	if err != nil {
		m.status = err.Error()
		return
	}
	text := normalizePractice(string(raw))
	switch msg.edit.kind {
	case "new":
		m.finishNew(msg.edit.name, text)
	case "edit":
		m.finishEdit(msg.edit, text)
	default:
		m.status = "cancelled"
	}
}

func normalizePractice(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSuffix(text, "\n")
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return text
}

func (m *model) finishNew(name, text string) {
	if text == "" {
		m.status = "cancelled"
		return
	}
	body := course.PageBody(name, text)
	slug, err := persistNewCourse(name, body)
	if err != nil {
		m.status = err.Error()
		return
	}
	parsed, ok := course.ParseMarkdown(slug, body)
	if !ok {
		m.status = "course would be empty"
		return
	}
	parsed.SourceFile = slug
	parsed.Body = body
	m.cfg.Courses = append(m.cfg.Courses, parsed)
	m.sortCourses()
	m.rebuildPages()
	m.selectStoredPage(slug, 0, 0, 0)
	m.status = "created"
}

func (m *model) finishEdit(edit scriptEdit, text string) {
	if text == edit.before {
		m.status = "unchanged"
		return
	}
	if text == "" {
		m.status = "missing text"
		return
	}
	if edit.index < 0 || edit.index >= len(m.pages) {
		m.status = "nothing to edit"
		return
	}
	p := m.pages[edit.index]
	item, ok := m.courseAt(p.courseIndex)
	if !ok {
		return
	}
	body, err := course.SetLessonText(item.Body, p.partIndex, p.sectionIndex, p.lessonIndex, text)
	if err != nil {
		m.status = err.Error()
		return
	}
	if err := m.storeCourse(p.courseIndex, item.SourceFile, item.Name, body); err != nil {
		m.status = err.Error()
		return
	}
	m.rebuildPages()
	m.selectStoredPage(item.SourceFile, p.partIndex, p.sectionIndex, p.lessonIndex)
	m.status = "saved"
}

func (m *model) courseNameTaken(name string) bool {
	slug := sessionSlug(name)
	for _, item := range m.cfg.Courses {
		if item.SourceFile == slug {
			return true
		}
	}
	return false
}

func sessionSlug(name string) string {
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	base = strings.TrimPrefix(base, "course_")
	base = strings.ToLower(strings.TrimSpace(base))
	base = strings.ReplaceAll(base, "_", "-")
	base = strings.ReplaceAll(base, " ", "-")
	if base == "" {
		return "untitled"
	}
	return base
}
