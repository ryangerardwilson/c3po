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
	if len(cfg.InitialLessons) > 0 {
		m.sequenceName = cfg.InitialSequenceName
		m.lessons = cfg.InitialLessons
		m.lessonIndex = clamp(cfg.InitialLessonIndex, 0, len(m.lessons)-1)
		m.screen = screenDoc
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
	if key == "?" {
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
	switch m.screen {
	case screenCourses:
		if len(m.cfg.Courses) == 0 {
			return
		}
		m.courseIndex = m.selected
		c := m.cfg.Courses[m.courseIndex]
		if len(c.Parts) == 1 && c.Parts[0].Name == "Main" && len(c.Parts[0].Sections) == 1 && c.Parts[0].Sections[0].Name == "Main" {
			m.openSequence(c.Name, c.Parts[0].Sections[0].Lessons, 0)
		} else {
			m.screen = screenParts
			m.selected = 0
		}
	case screenParts:
		c := m.cfg.Courses[m.courseIndex]
		m.partIndex = m.selected
		part := c.Parts[m.partIndex]
		if len(part.Sections) == 1 && part.Sections[0].Name == "Main" {
			m.openSequence(c.Name+": "+part.Name, part.Sections[0].Lessons, 0)
		} else {
			m.screen = screenSections
			m.selected = 0
		}
	case screenSections:
		c := m.cfg.Courses[m.courseIndex]
		part := c.Parts[m.partIndex]
		m.sectionIndex = m.selected
		section := part.Sections[m.sectionIndex]
		m.openSequence(c.Name+": "+part.Name+": "+section.Name, section.Lessons, 0)
	}
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
	switch m.screen {
	case screenCourses:
		return len(m.cfg.Courses)
	case screenParts:
		if len(m.cfg.Courses) == 0 {
			return 0
		}
		return len(m.cfg.Courses[m.courseIndex].Parts)
	case screenSections:
		if len(m.cfg.Courses) == 0 {
			return 0
		}
		return len(m.cfg.Courses[m.courseIndex].Parts[m.partIndex].Sections)
	default:
		return 0
	}
}

func (m model) itemName(index int) string {
	switch m.screen {
	case screenCourses:
		return m.cfg.Courses[index].Name
	case screenParts:
		return m.cfg.Courses[m.courseIndex].Parts[index].Name
	case screenSections:
		return m.cfg.Courses[m.courseIndex].Parts[m.partIndex].Sections[index].Name
	default:
		return ""
	}
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
		"Menus:",
		"  j/k move, l/enter open, h/esc back, q quit",
		"  n creates a course, part, or section",
		"  ,rn renames the selected item, dd deletes it",
		"",
		"Doc mode:",
		"  j/k scroll, ctrl+j/ctrl+k half-page, n/p next/previous lesson",
		"  t or r starts line typing, h returns to courses",
		"",
		"Type mode:",
		"  type over the visible line exactly, enter accepts it, ctrl+r resets, esc returns",
		"",
		"Press ? to return.",
	}
	return strings.Join(lines, "\n")
}

func (m model) listTitle() string {
	switch m.screen {
	case screenParts:
		return m.cfg.Courses[m.courseIndex].Name
	case screenSections:
		c := m.cfg.Courses[m.courseIndex]
		return c.Name + " > " + c.Parts[m.partIndex].Name
	default:
		return "nlp"
	}
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
	slug := ""
	cursor := "courses 0"
	switch screen {
	case screenCourses:
		if m.selected >= 0 && m.selected < len(m.cfg.Courses) {
			slug = m.cfg.Courses[m.selected].SourceFile
		}
		cursor = fmt.Sprintf("courses %d", m.selected)
	case screenParts:
		slug = m.courseSlug()
		cursor = fmt.Sprintf("parts %d %d", m.courseIndex, m.selected)
	case screenSections:
		slug = m.courseSlug()
		cursor = fmt.Sprintf("sections %d %d %d", m.courseIndex, m.partIndex, m.selected)
	case screenDoc:
		slug = m.courseSlug()
		cursor = fmt.Sprintf("doc %d %d %d %d", m.courseIndex, m.partIndex, m.sectionIndex, m.lessonIndex)
	case screenType:
		slug = m.courseSlug()
		cursor = fmt.Sprintf("type %d %d %d %d", m.courseIndex, m.partIndex, m.sectionIndex, m.lessonIndex)
	}
	_ = m.session.Place(slug, cursor)
}

func (m *model) restore(slug, cursor string) {
	fields := strings.Fields(cursor)
	if len(fields) == 0 || len(m.cfg.Courses) == 0 {
		return
	}
	need := map[string]int{"courses": 1, "parts": 2, "sections": 3, "doc": 4, "type": 4}
	count, ok := need[fields[0]]
	if !ok || len(fields)-1 < count {
		return
	}
	nums := make([]int, count)
	for i, field := range fields[1 : 1+count] {
		n, err := strconv.Atoi(field)
		if err != nil {
			return
		}
		nums[i] = n
	}
	if slug != "" {
		for i, item := range m.cfg.Courses {
			if item.SourceFile == slug {
				nums[0] = i
				break
			}
		}
	}
	if fields[0] == "courses" {
		m.screen = screenCourses
		m.selected = clamp(nums[0], 0, len(m.cfg.Courses)-1)
		m.courseIndex = m.selected
		return
	}
	if nums[0] < 0 || nums[0] >= len(m.cfg.Courses) {
		return
	}
	m.courseIndex = nums[0]
	item := m.cfg.Courses[m.courseIndex]
	switch fields[0] {
	case "parts":
		if len(item.Parts) == 0 {
			return
		}
		m.screen = screenParts
		m.selected = clamp(nums[1], 0, len(item.Parts)-1)
		m.partIndex = m.selected
	case "sections":
		if nums[1] < 0 || nums[1] >= len(item.Parts) || len(item.Parts[nums[1]].Sections) == 0 {
			return
		}
		m.partIndex = nums[1]
		m.screen = screenSections
		m.selected = clamp(nums[2], 0, len(item.Parts[nums[1]].Sections)-1)
		m.sectionIndex = m.selected
	case "doc", "type":
		if nums[1] < 0 || nums[1] >= len(item.Parts) {
			return
		}
		part := item.Parts[nums[1]]
		if nums[2] < 0 || nums[2] >= len(part.Sections) {
			return
		}
		section := part.Sections[nums[2]]
		m.partIndex = nums[1]
		m.sectionIndex = nums[2]
		name := item.Name
		if part.Name != "Main" {
			name += ": " + part.Name
		}
		if section.Name != "Main" {
			name += ": " + section.Name
		}
		m.openSequence(name, section.Lessons, nums[3])
		if fields[0] == "type" {
			m.startTyping()
		}
	}
}
