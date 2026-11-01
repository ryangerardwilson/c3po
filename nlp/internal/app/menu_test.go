package app

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ryangerardwilson/c3po/nlp/internal/course"
	"github.com/ryangerardwilson/c3po/nlp/internal/state"
)

const menuBody = `# Alpha

## Part I: One

### Section 1: A

#### Lesson 1: Start

    one

## Part II: Two

### Section 1: B

#### Lesson 1: Next

    two
`

func TestMenuCreateRenameAndDelete(t *testing.T) {
	var saved []string
	var created []string
	var removed []string
	persistCourse = func(slug, name, body string) error {
		saved = append(saved, slug+"|"+name+"|"+body)
		return nil
	}
	persistNewCourse = func(name, body string) (string, error) {
		created = append(created, name)
		if name == "Unix" {
			return "", errString("course already exists")
		}
		return "widgets", nil
	}
	removeCourse = func(slug string) error {
		removed = append(removed, slug)
		return nil
	}
	vimOpens := 0
	prevFind, prevStart := findVim, startVim
	findVim = func() (string, error) { return "vim", nil }
	startVim = func(path string, done func(error) tea.Msg) tea.Cmd {
		vimOpens++
		if err := os.WriteFile(path, []byte("the lord is my shepherd\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return func() tea.Msg { return done(nil) }
	}
	t.Cleanup(func() {
		persistCourse = state.SaveCourse
		persistNewCourse = state.CreateCourse
		removeCourse = state.DeleteStoredCourse
		findVim = prevFind
		startVim = prevStart
	})

	parsed, ok := course.ParseMarkdown("alpha", menuBody)
	if !ok {
		t.Fatal("fixture did not parse")
	}
	m := newModel(Config{Courses: []course.Course{parsed}})
	m.width = 80
	m.height = 24
	m.screen = screenCourses
	m.selected = 0

	view := stripANSI(m.View())
	if strings.Contains(view, "? help") || strings.Contains(view, "bookmark") || strings.Contains(view, "j/k move") {
		t.Fatalf("list view shows a shortcut string:\n%s", view)
	}

	m.screen = screenDoc
	doc := stripANSI(m.View())
	if strings.Contains(doc, "? help") || strings.Contains(doc, "bookmark") || strings.Contains(doc, "j/k scroll") {
		t.Fatalf("doc view shows a shortcut string:\n%s", doc)
	}
	m.screen = screenCourses

	armed := press(m, runeKey('d'))
	if armed.status != "press d again to delete" {
		t.Fatalf("status = %q", armed.status)
	}
	if len(armed.cfg.Courses) != 1 {
		t.Fatal("single d deleted a course")
	}
	moved := press(armed, runeKey('j'))
	if moved.deleteArmed {
		t.Fatal("movement left delete armed")
	}

	if join := strings.Join(pageNames(m), ","); join != "Next,Start" {
		t.Fatalf("pages = %s", join)
	}
	if strings.Contains(view, "Part I") || strings.Contains(view, "Section") {
		t.Fatalf("list still shows parts or sections:\n%s", view)
	}

	renamed := press(m, runeKey(','), runeKey('r'), runeKey('n'), tea.KeyMsg{Type: tea.KeyBackspace}, runeKey('x'), tea.KeyMsg{Type: tea.KeyEnter})
	if renamed.status != "renamed" {
		t.Fatalf("rename status = %q", renamed.status)
	}
	if renamed.pages[renamed.selected].name != "Nexx" {
		t.Fatalf("name = %q", renamed.pages[renamed.selected].name)
	}
	if renamed.cfg.Courses[0].Name != "Alpha" {
		t.Fatalf("course name changed to %q", renamed.cfg.Courses[0].Name)
	}
	if len(saved) != 1 {
		t.Fatalf("saved = %#v", saved)
	}

	leader := press(m, runeKey(','), runeKey('n'))
	if leader.status != "unknown leader action" || leader.inputKind != "" {
		t.Fatalf("leader n created an item: %#v", leader.status)
	}

	made, cmd := lastCmd(m, runeKey('n'), runeKey('P'), runeKey('r'), runeKey('a'), runeKey('y'), runeKey('e'), runeKey('r'), tea.KeyMsg{Type: tea.KeyEnter})
	made = applyCmd(t, made, cmd)
	if made.status != "created" {
		t.Fatalf("create status = %q", made.status)
	}
	if vimOpens != 1 {
		t.Fatalf("vim opens = %d", vimOpens)
	}
	idx := courseIndexBySlug(made.cfg.Courses, "widgets")
	if idx < 0 {
		t.Fatal("missing created course")
	}
	if !strings.Contains(made.cfg.Courses[idx].Body, "the lord is my shepherd") || strings.Contains(made.cfg.Courses[idx].Body, "...") {
		t.Fatalf("created body = %q", made.cfg.Courses[idx].Body)
	}
	if len(created) != 1 || created[0] != "Prayer" {
		t.Fatalf("created = %#v", created)
	}
	if made.pages[made.selected].name != "Prayer" {
		t.Fatalf("selected = %q", made.pages[made.selected].name)
	}
	opened := press(made, runeKey('l'))
	if opened.screen != screenType || opened.sequenceName != "Prayer" {
		t.Fatalf("open screen %v name %q", opened.screen, opened.sequenceName)
	}
	back := press(opened, tea.KeyMsg{Type: tea.KeyEsc})
	if back.screen != screenCourses {
		t.Fatalf("esc screen %v", back.screen)
	}

	deleted := press(m, runeKey('d'), runeKey('d'))
	if deleted.status != "deleted" {
		t.Fatalf("delete status = %q", deleted.status)
	}
	if strings.Join(pageNames(deleted), ",") != "Start" {
		t.Fatalf("pages after delete = %s", strings.Join(pageNames(deleted), ","))
	}
	if len(removed) != 0 {
		t.Fatalf("deleting one page removed the course %#v", removed)
	}

	help, _ := m.Update(runeKey('?'))
	shown := help.(model)
	if shown.screen != screenHelp {
		t.Fatal("? did not open help")
	}
	helpText := stripANSI(shown.View())
	for _, want := range []string{",rn", "dd", "vim", "e opens the selected session in vim", "Press ? to return."} {
		if !strings.Contains(helpText, want) {
			t.Fatalf("help missing %q:\n%s", want, helpText)
		}
	}
	if strings.Contains(helpText, "bookmark") || strings.Contains(helpText, "esc to return") {
		t.Fatalf("help still documents bookmarks or esc:\n%s", helpText)
	}
	stayed, cmd := shown.Update(runeKey('q'))
	if cmd != nil || stayed.(model).screen != screenHelp {
		t.Fatal("q closed help or quit")
	}
	stayed, cmd = shown.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil || stayed.(model).screen != screenHelp {
		t.Fatal("esc closed help")
	}
	closed, _ := shown.Update(runeKey('?'))
	if closed.(model).screen != screenCourses {
		t.Fatal("? did not close help")
	}
	_, cmd = shown.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}})
	if cmd == nil {
		t.Fatal("Q did not quit")
	}

	removedCourse := press(made, runeKey('d'), runeKey('d'))
	if removedCourse.status != "deleted" || len(removed) != 1 || removed[0] != "widgets" {
		t.Fatalf("removed = %#v status %q", removed, removedCourse.status)
	}
	if strings.Contains(strings.Join(pageNames(removedCourse), ","), "Prayer") {
		t.Fatalf("prayer remained %s", strings.Join(pageNames(removedCourse), ","))
	}
}

func pageNames(m model) []string {
	out := make([]string, len(m.pages))
	for i, page := range m.pages {
		out[i] = page.name
	}
	return out
}

func TestEnterOpensTyping(t *testing.T) {
	parsed, ok := course.ParseMarkdown("alpha", menuBody)
	if !ok {
		t.Fatal("fixture did not parse")
	}
	m := newModel(Config{Courses: []course.Course{parsed}})
	opened := press(m, runeKey('l'))
	if opened.screen != screenType || opened.sequenceName != "Next" {
		t.Fatalf("screen %v name %q", opened.screen, opened.sequenceName)
	}
	back := press(opened, tea.KeyMsg{Type: tea.KeyEsc})
	if back.screen != screenCourses || back.sequenceName != "Next" {
		t.Fatalf("back screen %v", back.screen)
	}
}

func TestTypeViewOmitsShortcutLine(t *testing.T) {
	m := testModel()
	m.width = 40
	m.height = 12
	m.startTyping()
	view := stripANSI(m.View())
	if strings.Contains(view, "ctrl+r") || strings.Contains(view, "esc doc") {
		t.Fatalf("type view shows a shortcut string:\n%s", view)
	}
}

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func press(m model, keys ...tea.KeyMsg) model {
	for _, key := range keys {
		next, _ := m.Update(key)
		m = next.(model)
	}
	return m
}

type errString string

func (e errString) Error() string { return string(e) }

func lastCmd(m model, keys ...tea.KeyMsg) (model, tea.Cmd) {
	var cmd tea.Cmd
	for _, key := range keys {
		next, c := m.Update(key)
		m = next.(model)
		cmd = c
	}
	return m, cmd
}

func applyCmd(t *testing.T, m model, cmd tea.Cmd) model {
	t.Helper()
	if cmd == nil {
		t.Fatal("missing vim command")
	}
	next, nextCmd := m.Update(cmd())
	if nextCmd != nil {
		t.Fatal("vim result returned another command")
	}
	return next.(model)
}

func TestEditSessionInVim(t *testing.T) {
	var saved []string
	prevFind, prevStart := findVim, startVim
	findVim = func() (string, error) { return "vim", nil }
	persistCourse = func(slug, name, body string) error {
		saved = append(saved, slug+"|"+name+"|"+body)
		return nil
	}
	t.Cleanup(func() {
		persistCourse = state.SaveCourse
		findVim = prevFind
		startVim = prevStart
	})

	parsed, ok := course.ParseMarkdown("alpha", menuBody)
	if !ok {
		t.Fatal("fixture did not parse")
	}
	m := newModel(Config{Courses: []course.Course{parsed}})
	if len(m.pages) == 0 || m.pages[0].name != "Next" {
		t.Fatalf("selected %#v", pageNames(m))
	}
	startVim = func(path string, done func(error) tea.Msg) tea.Cmd {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(raw) != "two\n" {
			t.Fatalf("vim buffer = %q", raw)
		}
		if err := os.WriteFile(path, []byte("two\nmore\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return func() tea.Msg { return done(nil) }
	}
	edited, cmd := lastCmd(m, runeKey('e'))
	edited = applyCmd(t, edited, cmd)
	if edited.status != "saved" || edited.cfg.Courses[0].Name != "Alpha" {
		t.Fatalf("status %q name %q", edited.status, edited.cfg.Courses[0].Name)
	}
	if len(saved) != 1 {
		t.Fatalf("saved = %#v", saved)
	}
	body := edited.cfg.Courses[0].Body
	got, err := course.LessonText(body, m.pages[0].partIndex, m.pages[0].sectionIndex, m.pages[0].lessonIndex)
	if err != nil || got != "two\nmore" {
		t.Fatalf("edited text = %q err=%v", got, err)
	}
	other, err := course.LessonText(body, m.pages[1].partIndex, m.pages[1].sectionIndex, m.pages[1].lessonIndex)
	if err != nil || other != "one" {
		t.Fatalf("other text = %q err=%v", other, err)
	}

	startVim = func(path string, done func(error) tea.Msg) tea.Cmd {
		return func() tea.Msg { return done(nil) }
	}
	same, cmd := lastCmd(edited, runeKey('e'))
	same = applyCmd(t, same, cmd)
	if same.status != "unchanged" || len(saved) != 1 {
		t.Fatalf("unchanged status %q saved %d", same.status, len(saved))
	}

	startVim = func(path string, done func(error) tea.Msg) tea.Cmd {
		if err := os.WriteFile(path, []byte("\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return func() tea.Msg { return done(nil) }
	}
	wiped, cmd := lastCmd(edited, runeKey('e'))
	wiped = applyCmd(t, wiped, cmd)
	if wiped.status != "missing text" || len(saved) != 1 {
		t.Fatalf("blank status %q saved %d", wiped.status, len(saved))
	}

	typed := press(m, runeKey('l'), runeKey('e'))
	if typed.screen != screenType || len(typed.typeInputs) == 0 || typed.typeInputs[0] != "e" {
		t.Fatalf("type screen %v input %#v", typed.screen, typed.typeInputs)
	}
	leader := press(m, runeKey(','), runeKey('e'))
	if leader.status != "unknown leader action" {
		t.Fatalf("leader e = %q", leader.status)
	}
}

func TestNewSessionVimCancelAndMissing(t *testing.T) {
	var created []string
	prevFind, prevStart := findVim, startVim
	persistNewCourse = func(name, body string) (string, error) {
		created = append(created, name+"|"+body)
		return "prayer", nil
	}
	findVim = func() (string, error) { return "vim", nil }
	startVim = func(path string, done func(error) tea.Msg) tea.Cmd {
		return func() tea.Msg { return done(nil) }
	}
	t.Cleanup(func() {
		persistNewCourse = state.CreateCourse
		findVim = prevFind
		startVim = prevStart
	})
	parsed, ok := course.ParseMarkdown("alpha", menuBody)
	if !ok {
		t.Fatal("fixture did not parse")
	}
	m := newModel(Config{Courses: []course.Course{parsed}})
	cancelled, cmd := lastCmd(m, runeKey('n'), runeKey('P'), tea.KeyMsg{Type: tea.KeyEnter})
	cancelled = applyCmd(t, cancelled, cmd)
	if cancelled.status != "cancelled" || len(created) != 0 || len(cancelled.pages) != len(m.pages) {
		t.Fatalf("status %q created %#v", cancelled.status, created)
	}

	findVim = func() (string, error) { return "", errString("vim is not on PATH") }
	missing, cmd := lastCmd(m, runeKey('n'), runeKey('P'), tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || missing.status != "vim is not on PATH" || len(created) != 0 {
		t.Fatalf("status %q cmd %v created %#v", missing.status, cmd != nil, created)
	}

	findVim = func() (string, error) { return "vim", nil }
	dup, cmd := lastCmd(m, runeKey('n'), runeKey('A'), runeKey('l'), runeKey('p'), runeKey('h'), runeKey('a'), tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || dup.status != "course already exists" || len(created) != 0 {
		t.Fatalf("dup status %q cmd %v created %#v", dup.status, cmd != nil, created)
	}
}

func TestQuestionMarkTypesDuringTouchTyping(t *testing.T) {
	parsed, ok := course.ParseMarkdown("alpha", menuBody)
	if !ok {
		t.Fatal("fixture did not parse")
	}
	m := newModel(Config{Courses: []course.Course{parsed}})
	opened := press(m, runeKey('l'))
	if opened.screen != screenType {
		t.Fatalf("screen %v", opened.screen)
	}
	typed := press(opened, runeKey('?'))
	if typed.screen != screenType {
		t.Fatalf("help opened, screen %v", typed.screen)
	}
	if len(typed.typeInputs) == 0 || typed.typeInputs[0] != "?" {
		t.Fatalf("typed %#v", typed.typeInputs)
	}
}
