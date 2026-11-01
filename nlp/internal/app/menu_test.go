package app

import (
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
	t.Cleanup(func() {
		persistCourse = state.SaveCourse
		persistNewCourse = state.CreateCourse
		removeCourse = state.DeleteStoredCourse
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

	made := press(m, runeKey('n'), runeKey('P'), runeKey('r'), runeKey('a'), runeKey('y'), runeKey('e'), runeKey('r'), tea.KeyMsg{Type: tea.KeyEnter})
	if made.status != "created" {
		t.Fatalf("create status = %q", made.status)
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
	for _, want := range []string{",rn", "dd", "Press ? to return."} {
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
