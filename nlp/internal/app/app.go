package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ryangerardwilson/c3po/internal/tuisession"
	"github.com/ryangerardwilson/c3po/nlp/internal/course"
)

type Config struct {
	Courses             []course.Course
	InitialSequenceName string
	InitialLessons      []course.Lesson
	InitialLessonIndex  int
}

type screen int

const (
	screenCourses screen = iota
	screenParts
	screenSections
	screenDoc
	screenType
	screenHelp
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("238"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("80"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

type model struct {
	cfg     Config
	session *tuisession.Session

	screen screen
	before screen

	courseIndex  int
	partIndex    int
	sectionIndex int
	selected     int
	pages        []page
	scroll       int

	sequenceName string
	lessons      []course.Lesson
	lessonIndex  int

	typeLines      []string
	typeTargets    []string
	typeSkip       []bool
	typeInputs     []string
	typeLine       int
	typed          string
	typeDone       bool
	cursorShown    bool
	cursorBlinking bool

	inputKind   string
	input       string
	leader      bool
	leaderSeq   string
	deleteArmed bool
	newLabel    string

	status string
	width  int
	height int
}

func Run(cfg Config) int {
	return RunSession(cfg, nil)
}

func RunSession(cfg Config, sess *tuisession.Session) int {
	m := newModel(cfg)
	m.session = sess
	if sess != nil {
		if body, ok := sess.Restored(); ok {
			m.restore(body.Path, body.Cursor)
		}
	}
	program := tea.NewProgram(m, tea.WithAltScreen())
	final, err := program.Run()
	if done, ok := final.(model); ok {
		done.remember()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func newModel(cfg Config) model {
	m := model{
		cfg:         cfg,
		screen:      screenCourses,
		status:      "ready",
		width:       80,
		height:      24,
		cursorShown: true,
	}
	m.rebuildPages()
	if len(cfg.InitialLessons) > 0 {
		m.sequenceName = cfg.InitialSequenceName
		m.lessons = cfg.InitialLessons
		m.lessonIndex = clamp(cfg.InitialLessonIndex, 0, len(m.lessons)-1)
		m.startTyping()
	}
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.updateMsg(msg)
	switch msg.(type) {
	case cursorBlinkMsg, tea.WindowSizeMsg:
		return next, cmd
	}
	if updated, ok := next.(model); ok {
		updated.remember()
		return updated, cmd
	}
	return next, cmd
}

func (m model) updateMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case cursorBlinkMsg:
		if m.screen != screenType || m.typeDone {
			m.cursorShown = true
			m.cursorBlinking = false
			return m, nil
		}
		m.cursorBlinking = true
		m.cursorShown = !m.cursorShown
		return m, cursorBlinkCmd()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	case scriptSavedMsg:
		m.finishScript(msg)
		return m, nil
	default:
		return m, nil
	}
}

func (m model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.inputKind != "" {
		return m.updateInputKey(msg)
	}
	if key == "ctrl+c" || key == "Q" {
		return m, tea.Quit
	}
	if key == "?" && m.screen != screenType {
		m.leader = false
		m.leaderSeq = ""
		m.deleteArmed = false
		if m.screen == screenHelp {
			m.screen = m.before
		} else {
			m.before = m.screen
			m.screen = screenHelp
		}
		return m, nil
	}
	if m.screen == screenHelp {
		return m, nil
	}
	if m.menuScreen() && m.leader {
		return m.updateLeader(key)
	}
	if m.menuScreen() && m.deleteArmed && key != "d" {
		m.deleteArmed = false
		m.status = "ready"
	}

	switch m.screen {
	case screenCourses, screenParts, screenSections:
		return m.updateListKey(key)
	case screenDoc:
		return m.updateDocKey(key)
	case screenType:
		return m.updateTypeKey(msg)
	}
	return m, nil
}

func (m model) updateListKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "up", "k":
		m.moveSelection(-1)
	case "down", "j":
		m.moveSelection(1)
	case "left", "h", "esc":
		m.goBack()
	case "right", "l", "enter":
		m.activate()
		if m.screen == screenType && !m.cursorBlinking {
			m.cursorBlinking = true
			return m, cursorBlinkCmd()
		}
	case ",":
		m.leader = true
		m.leaderSeq = ""
		m.deleteArmed = false
		m.status = ",rn rename"
	case "d":
		if m.deleteArmed {
			m.deleteArmed = false
			m.deleteSelected()
		} else {
			m.deleteArmed = true
			m.status = "press d again to delete"
		}
	case "n":
		m.leader = false
		m.leaderSeq = ""
		m.beginNew()
	case "e":
		if m.screen != screenCourses {
			return m, nil
		}
		return m, m.editSelected()
	}
	return m, nil
}

func (m model) updateDocKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "left", "h", "esc":
		m.screen = screenCourses
		m.selected = m.courseIndex
		m.scroll = 0
	case "up", "k":
		if m.scroll > 0 {
			m.scroll--
		}
	case "down", "j":
		m.scroll++
	case "ctrl+k", "pgup":
		m.scroll -= max(1, m.height/2)
		if m.scroll < 0 {
			m.scroll = 0
		}
	case "ctrl+j", "pgdown":
		m.scroll += max(1, m.height/2)
	case "n", "right", "l":
		m.nextLesson()
	case "p":
		m.prevLesson()
	case "t", "r":
		m.startTyping()
		if !m.cursorBlinking {
			m.cursorBlinking = true
			return m, cursorBlinkCmd()
		}
		return m, nil
	}
	return m, nil
}

func (m *model) moveSelection(delta int) {
	count := m.itemCount()
	if count == 0 {
		m.selected = 0
		return
	}
	m.selected = (m.selected + delta + count) % count
}

func (m *model) activate() {
	if m.screen != screenCourses {
		return
	}
	m.openPage(m.selected)
}

func (m *model) goBack() {
	switch m.screen {
	case screenParts:
		m.screen = screenCourses
		m.selected = m.courseIndex
	case screenSections:
		m.screen = screenParts
		m.selected = m.partIndex
	}
}

func (m *model) openSequence(name string, lessons []course.Lesson, index int) {
	m.sequenceName = name
	m.lessons = lessons
	m.lessonIndex = clamp(index, 0, len(lessons)-1)
	m.scroll = 0
	m.screen = screenDoc
	m.status = "doc mode"
}

func (m *model) nextLesson() {
	if m.lessonIndex < len(m.lessons)-1 {
		m.lessonIndex++
		m.scroll = 0
		m.status = m.currentLesson().Name
	}
}

func (m *model) prevLesson() {
	if m.lessonIndex > 0 {
		m.lessonIndex--
		m.scroll = 0
		m.status = m.currentLesson().Name
	}
}

func (m model) currentLesson() course.Lesson {
	if len(m.lessons) == 0 {
		return course.Lesson{Name: "No lessons", Content: ""}
	}
	return m.lessons[clamp(m.lessonIndex, 0, len(m.lessons)-1)]
}

func (m model) itemCount() int {
	if m.screen == screenCourses {
		return len(m.pages)
	}
	return 0
}

func (m model) itemName(index int) string {
	if m.screen != screenCourses || index < 0 || index >= len(m.pages) {
		return ""
	}
	return m.pages[index].name
}

func (m model) View() string {
	var view string
	switch m.screen {
	case screenDoc:
		view = m.renderDoc()
	case screenType:
		view = m.renderType()
	case screenHelp:
		view = m.renderHelp()
	default:
		view = m.renderList()
	}
	return fitToHeight(view, m.height)
}

func (m model) renderList() string {
	var out strings.Builder
	out.WriteString(titleStyle.Render(m.listTitle()))
	out.WriteString("\n\n")
	count := m.itemCount()
	if count == 0 {
		out.WriteString(mutedStyle.Render("No items."))
	} else {
		visible := max(6, m.height-6)
		start := 0
		if m.selected >= visible {
			start = m.selected - visible + 1
		}
		end := min(count, start+visible)
		for i := start; i < end; i++ {
			prefix := "  "
			if i == m.selected {
				prefix = "> "
			}
			line := prefix + truncateMiddle(m.itemName(i), m.width-2)
			if i == m.selected {
				out.WriteString(selectedStyle.Render(line))
			} else {
				out.WriteString(line)
			}
			out.WriteString("\n")
		}
	}
	out.WriteString("\n")
	out.WriteString(statusStyle.Render(m.status))
	return out.String()
}

func (m model) renderDoc() string {
	lesson := m.currentLesson()
	lines := strings.Split(strings.TrimRight(lesson.Content, "\n"), "\n")
	visible := max(5, m.height-6)
	if m.scroll > max(0, len(lines)-visible) {
		m.scroll = max(0, len(lines)-visible)
	}
	end := min(len(lines), m.scroll+visible)

	var out strings.Builder
	out.WriteString(titleStyle.Render(truncateMiddle(m.sequenceName, m.width)))
	out.WriteString("\n")
	out.WriteString(statusStyle.Render(fmt.Sprintf("%s  [%d/%d]", lesson.Name, m.lessonIndex+1, max(1, len(m.lessons)))))
	out.WriteString("\n\n")
	for _, line := range lines[m.scroll:end] {
		out.WriteString(truncateMiddle(line, m.width))
		out.WriteString("\n")
	}
	out.WriteString("\n")
	out.WriteString(statusStyle.Render(m.status))
	return out.String()
}

func fitToHeight(view string, height int) string {
	if height <= 0 {
		return view
	}
	lines := strings.Split(view, "\n")
	if len(lines) > height {
		return strings.Join(lines[:height], "\n")
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m model) renderHelp() string {
	lines := []string{
		titleStyle.Render("nlp help"),
		"",
		"Sessions:",
		"  j/k move, enter opens the page, q quits",
		"  n creates a session, then vim edits its text",
		"  e opens the selected session in vim",
		"  ,rn renames the selected session, dd deletes it",
		"",
		"Type mode:",
		"  type over the visible line exactly, including ?, enter accepts it, ctrl+r resets, esc returns",
		"",
		"Press ? to return.",
	}
	return strings.Join(lines, "\n")
}

func (m model) listTitle() string {
	return "nlp"
}

func trimLastRune(value string) string {
	if value == "" {
		return value
	}
	_, size := utf8.DecodeLastRuneInString(value)
	return value[:len(value)-size]
}

func truncateMiddle(value string, limit int) string {
	value = strings.TrimRightFunc(value, unicode.IsSpace)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit < 5 {
		return value[:limit]
	}
	half := (limit - 3) / 2
	return value[:half] + "..." + value[len(value)-(limit-3-half):]
}

func clamp(value, low, high int) int {
	if high < low {
		return low
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m model) courseSlug() string {
	if m.courseIndex < 0 || m.courseIndex >= len(m.cfg.Courses) {
		return ""
	}
	return m.cfg.Courses[m.courseIndex].SourceFile
}

func (m model) remember() {
	if m.session == nil {
		return
	}
	screen := m.screen
	if screen == screenHelp {
		screen = m.before
	}
	selected := m.selected
	if selected < 0 || selected >= len(m.pages) {
		selected = 0
	}
	slug := ""
	if len(m.pages) > 0 && m.pages[selected].courseIndex >= 0 && m.pages[selected].courseIndex < len(m.cfg.Courses) {
		slug = m.cfg.Courses[m.pages[selected].courseIndex].SourceFile
	}
	kind := "pages"
	if screen == screenType {
		kind = "type"
	}
	_ = m.session.Place(slug, fmt.Sprintf("%s %d", kind, selected))
}

func (m *model) restore(slug, cursor string) {
	m.rebuildPages()
	if len(m.pages) == 0 {
		return
	}
	fields := strings.Fields(cursor)
	if len(fields) == 0 {
		return
	}
	switch fields[0] {
	case "pages", "courses":
		m.screen = screenCourses
		m.selected = m.pageIndex(slug, numberAt(fields, 1))
		if m.selected >= 0 && m.selected < len(m.pages) {
			m.courseIndex = m.pages[m.selected].courseIndex
		}
	case "parts", "sections":
		m.screen = screenCourses
		m.selected = m.pageIndex(slug, 0)
		if m.selected >= 0 && m.selected < len(m.pages) {
			m.courseIndex = m.pages[m.selected].courseIndex
		}
	case "doc", "type":
		if len(fields) >= 5 {
			courseIndex := numberAt(fields, 1)
			if slug != "" {
				if found := courseIndexBySlug(m.cfg.Courses, slug); found >= 0 {
					courseIndex = found
				}
			}
			m.selected = m.pageByLesson(courseIndex, numberAt(fields, 2), numberAt(fields, 3), numberAt(fields, 4))
		} else {
			m.selected = m.pageIndex(slug, numberAt(fields, 1))
		}
		m.openPage(m.selected)
	}
}

func numberAt(fields []string, index int) int {
	if index < 0 || index >= len(fields) {
		return 0
	}
	n, err := strconv.Atoi(fields[index])
	if err != nil {
		return 0
	}
	return n
}
