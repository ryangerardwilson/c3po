package app

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ryangerardwilson/c3po/nlp/internal/course"
	"github.com/ryangerardwilson/c3po/nlp/internal/state"
)

var (
	persistCourse    = state.SaveCourse
	persistNewCourse = state.CreateCourse
	removeCourse     = state.DeleteStoredCourse
)

func (m model) menuScreen() bool {
	return m.screen == screenCourses || m.screen == screenParts || m.screen == screenSections
}

func (m model) updateInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputKind = ""
		m.input = ""
		m.status = "cancelled"
		return m, nil
	case "enter":
		m.commitInput()
		return m, nil
	case "backspace", "ctrl+h":
		m.input = trimLastRune(m.input)
		m.paintInput()
		return m, nil
	case "ctrl+w":
		m.input = trimPreviousWord(m.input)
		m.paintInput()
		return m, nil
	default:
		if len(msg.Runes) > 0 && !strings.ContainsAny(string(msg.Runes), "\r\n") {
			m.input += string(msg.Runes)
			m.paintInput()
		}
		return m, nil
	}
}

func (m model) updateLeader(key string) (tea.Model, tea.Cmd) {
	m.leaderSeq += key
	switch {
	case m.leaderSeq == "rn":
		m.leader = false
		m.leaderSeq = ""
		m.beginRename()
	case strings.HasPrefix("rn", m.leaderSeq):
		m.status = ",rn rename"
	default:
		m.leader = false
		m.leaderSeq = ""
		m.status = "unknown leader action"
	}
	return m, nil
}

func (m *model) beginRename() {
	if m.selected < 0 || m.selected >= m.itemCount() {
		m.status = "nothing to rename"
		return
	}
	name := m.itemName(m.selected)
	if name == "" {
		m.status = "nothing to rename"
		return
	}
	m.leader = false
	m.leaderSeq = ""
	m.deleteArmed = false
	m.inputKind = "rename"
	m.input = name
	m.paintInput()
}

func (m *model) beginNew() {
	switch m.screen {
	case screenCourses:
		m.newLabel = "new course"
	case screenParts:
		m.newLabel = "new part"
	case screenSections:
		m.newLabel = "new section"
	default:
		return
	}
	m.leader = false
	m.leaderSeq = ""
	m.deleteArmed = false
	m.inputKind = "new"
	m.input = ""
	m.paintInput()
}

func (m *model) paintInput() {
	label := "rename"
	if m.inputKind == "new" {
		label = m.newLabel
	}
	m.status = strings.TrimSpace(label + " " + m.input)
}

func (m *model) commitInput() {
	value := strings.TrimSpace(m.input)
	kind := m.inputKind
	m.inputKind = ""
	m.input = ""
	if value == "" {
		m.status = "missing name"
		return
	}
	if kind == "rename" {
		m.commitRename(value)
		return
	}
	m.commitNew(value)
}

func (m *model) commitRename(name string) {
	switch m.screen {
	case screenCourses:
		m.renameCourse(m.selected, name)
	case screenParts:
		m.renamePart(m.selected, name)
	case screenSections:
		m.renameSection(m.selected, name)
	}
}

func (m *model) commitNew(name string) {
	switch m.screen {
	case screenCourses:
		m.createCourse(name)
	case screenParts:
		m.createPart(name)
	case screenSections:
		m.createSection(name)
	}
}

func (m *model) renameCourse(index int, name string) {
	item, ok := m.courseAt(index)
	if !ok {
		return
	}
	if name == item.Name {
		m.status = "unchanged"
		return
	}
	body, err := course.RenameCourse(item.Body, name)
	if err != nil {
		m.status = err.Error()
		return
	}
	if err := m.storeCourse(index, item.SourceFile, name, body); err != nil {
		m.status = err.Error()
		return
	}
	slug := item.SourceFile
	m.sortCourses()
	for i, stored := range m.cfg.Courses {
		if stored.SourceFile == slug {
			m.selected = i
			m.courseIndex = i
			break
		}
	}
	m.status = "renamed"
}

func (m *model) renamePart(index int, name string) {
	item, ok := m.courseAt(m.courseIndex)
	if !ok {
		return
	}
	body, err := course.RenamePart(item.Body, index, name)
	if err != nil {
		m.status = err.Error()
		return
	}
	if err := m.storeCourse(m.courseIndex, item.SourceFile, item.Name, body); err != nil {
		m.status = err.Error()
		return
	}
	m.status = "renamed"
}

func (m *model) renameSection(index int, name string) {
	item, ok := m.courseAt(m.courseIndex)
	if !ok {
		return
	}
	body, err := course.RenameSection(item.Body, m.partIndex, index, name)
	if err != nil {
		m.status = err.Error()
		return
	}
	if err := m.storeCourse(m.courseIndex, item.SourceFile, item.Name, body); err != nil {
		m.status = err.Error()
		return
	}
	m.status = "renamed"
}

func (m *model) createCourse(name string) {
	body := course.NewCourseBody(name)
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
	for i, item := range m.cfg.Courses {
		if item.SourceFile == slug {
			m.selected = i
			m.courseIndex = i
			break
		}
	}
	m.status = "created"
}

func (m *model) createPart(name string) {
	item, ok := m.courseAt(m.courseIndex)
	if !ok {
		return
	}
	after := m.selected
	body, err := course.InsertPart(item.Body, name, after, usesSections(item))
	if err != nil {
		m.status = err.Error()
		return
	}
	if err := m.storeCourse(m.courseIndex, item.SourceFile, item.Name, body); err != nil {
		m.status = err.Error()
		return
	}
	m.selected = after + 1
	if parts := m.cfg.Courses[m.courseIndex].Parts; m.selected >= len(parts) {
		m.selected = len(parts) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
	m.status = "created"
}

func (m *model) createSection(name string) {
	item, ok := m.courseAt(m.courseIndex)
	if !ok {
		return
	}
	after := m.selected
	body, err := course.InsertSection(item.Body, m.partIndex, after, name)
	if err != nil {
		m.status = err.Error()
		return
	}
	if err := m.storeCourse(m.courseIndex, item.SourceFile, item.Name, body); err != nil {
		m.status = err.Error()
		return
	}
	m.selected = after + 1
	sections := m.cfg.Courses[m.courseIndex].Parts[m.partIndex].Sections
	if m.selected >= len(sections) {
		m.selected = len(sections) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
	m.status = "created"
}

func (m *model) deleteSelected() {
	switch m.screen {
	case screenCourses:
		m.deleteCourse(m.selected)
	case screenParts:
		m.deletePart(m.selected)
	case screenSections:
		m.deleteSection(m.selected)
	}
}

func (m *model) deleteCourse(index int) {
	item, ok := m.courseAt(index)
	if !ok {
		return
	}
	if item.SourceFile == "" {
		m.status = "course has no stored text"
		return
	}
	if err := removeCourse(item.SourceFile); err != nil {
		m.status = err.Error()
		return
	}
	m.cfg.Courses = append(m.cfg.Courses[:index], m.cfg.Courses[index+1:]...)
	if m.selected >= len(m.cfg.Courses) {
		m.selected = len(m.cfg.Courses) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
	m.status = "deleted"
}

func (m *model) deletePart(index int) {
	item, ok := m.courseAt(m.courseIndex)
	if !ok {
		return
	}
	body, err := course.DeletePart(item.Body, index)
	if err != nil {
		m.status = err.Error()
		return
	}
	if err := m.storeCourse(m.courseIndex, item.SourceFile, item.Name, body); err != nil {
		m.status = err.Error()
		return
	}
	if m.selected >= len(m.cfg.Courses[m.courseIndex].Parts) {
		m.selected = len(m.cfg.Courses[m.courseIndex].Parts) - 1
	}
	m.status = "deleted"
}

func (m *model) deleteSection(index int) {
	item, ok := m.courseAt(m.courseIndex)
	if !ok {
		return
	}
	body, err := course.DeleteSection(item.Body, m.partIndex, index)
	if err != nil {
		m.status = err.Error()
		return
	}
	if err := m.storeCourse(m.courseIndex, item.SourceFile, item.Name, body); err != nil {
		m.status = err.Error()
		return
	}
	sections := m.cfg.Courses[m.courseIndex].Parts[m.partIndex].Sections
	if m.selected >= len(sections) {
		m.selected = len(sections) - 1
	}
	m.status = "deleted"
}

func (m *model) storeCourse(index int, slug, name, body string) error {
	parsed, ok := course.ParseMarkdown(slug, body)
	if !ok {
		return fmt.Errorf("course would be empty")
	}
	if err := persistCourse(slug, parsed.Name, body); err != nil {
		return err
	}
	parsed.SourceFile = slug
	parsed.Body = body
	if name != "" && parsed.Name == "" {
		parsed.Name = name
	}
	m.cfg.Courses[index] = parsed
	return nil
}

func (m *model) courseAt(index int) (course.Course, bool) {
	if index < 0 || index >= len(m.cfg.Courses) {
		m.status = "nothing selected"
		return course.Course{}, false
	}
	item := m.cfg.Courses[index]
	if strings.TrimSpace(item.Body) == "" {
		m.status = "course has no stored text"
		return course.Course{}, false
	}
	return item, true
}

func (m *model) sortCourses() {
	courses := m.cfg.Courses
	for i := 1; i < len(courses); i++ {
		item := courses[i]
		j := i
		for j > 0 && strings.ToLower(courses[j-1].Name) > strings.ToLower(item.Name) {
			courses[j] = courses[j-1]
			j--
		}
		courses[j] = item
	}
	m.cfg.Courses = courses
}

func usesSections(item course.Course) bool {
	for _, part := range item.Parts {
		if len(part.Sections) == 1 && part.Sections[0].Name == "Main" {
			continue
		}
		if len(part.Sections) > 0 {
			return true
		}
	}
	return false
}

func trimPreviousWord(value string) string {
	value = strings.TrimRightFunc(value, func(r rune) bool { return r == ' ' })
	for len(value) > 0 {
		_, size := lastRune(value)
		if size == 0 {
			break
		}
		r := value[len(value)-size]
		value = value[:len(value)-size]
		if r == ' ' {
			break
		}
	}
	return value
}

func lastRune(value string) (rune, int) {
	if value == "" {
		return 0, 0
	}
	return utf8.DecodeLastRuneInString(value)
}
