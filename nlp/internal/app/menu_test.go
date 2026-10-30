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

	renamed := press(m, runeKey(','), runeKey('r'), runeKey('n'), tea.KeyMsg{Type: tea.KeyBackspace}, runeKey('x'), tea.KeyMsg{Type: tea.KeyEnter})
	if renamed.status != "renamed" {
		t.Fatalf("rename status = %q", renamed.status)
	}
	if renamed.cfg.Courses[renamed.selected].Name != "Alphx" {
		t.Fatalf("name = %q", renamed.cfg.Courses[renamed.selected].Name)
	}
	if len(saved) != 1 {
		t.Fatalf("saved = %#v", saved)
	}

	leader := press(m, runeKey(','), runeKey('n'))
	if leader.status != "unknown leader action" || leader.inputKind != "" {
		t.Fatalf("leader n created an item: %#v", leader.status)
	}

	made := press(m, runeKey('n'), runeKey('W'), runeKey('i'), runeKey('d'), runeKey('g'), runeKey('e'), runeKey('t'), runeKey('s'), tea.KeyMsg{Type: tea.KeyEnter})
	if made.status != "created" {
		t.Fatalf("create status = %q", made.status)
	}
	if len(created) != 1 || created[0] != "Widgets" {
		t.Fatalf("created = %#v", created)
	}
	if made.cfg.Courses[made.selected].Name != "Widgets" {
		t.Fatalf("selected = %q", made.cfg.Courses[made.selected].Name)
	}

	parts := press(m, runeKey('l'))
	if parts.screen != screenParts {
		t.Fatalf("screen = %v", parts.screen)
	}
	deleted := press(parts, runeKey('d'), runeKey('d'))
	if deleted.status != "deleted" {
		t.Fatalf("delete status = %q", deleted.status)
	}
	if got := deleted.cfg.Courses[0].Parts[0].Name; got != "Part I: Two" {
		t.Fatalf("renumbered part = %q", got)
	}
	if strings.Contains(saved[len(saved)-1], "Part II:") {
		t.Fatalf("old part number remained:\n%s", saved[len(saved)-1])
	}

	sections := press(parts, runeKey('l'))
	if sections.screen != screenSections {
		t.Fatalf("screen = %v", sections.screen)
	}
	added := press(sections, runeKey('n'))
	typed := press(added, []tea.KeyMsg{
		runeKey('S'), runeKey('e'), runeKey('c'), runeKey('t'), runeKey('i'), runeKey('o'), runeKey('n'),
		runeKey(' '), runeKey('9'), runeKey(':'), runeKey(' '), runeKey('C'),
		{Type: tea.KeyEnter},
	}...)
	if typed.status != "created" {
		t.Fatalf("section status = %q", typed.status)
	}
	got := typed.cfg.Courses[0].Parts[0].Sections[typed.selected].Name
	if got != "Section 2: C" {
		t.Fatalf("new section = %q", got)
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

	removedCourse := press(m, runeKey('d'), runeKey('d'))
	if removedCourse.status != "deleted" || len(removed) != 1 || removed[0] != "alpha" {
		t.Fatalf("removed = %#v status %q", removed, removedCourse.status)
	}
}

func TestDocNAdvancesLesson(t *testing.T) {
	m := newModel(Config{
		Courses: []course.Course{{
			Name: "Course",
			Parts: []course.Part{{
				Name: "Main",
				Sections: []course.Section{{
					Name:    "Main",
					Lessons: []course.Lesson{{Name: "One", Content: "a"}, {Name: "Two", Content: "b"}},
				}},
			}},
		}},
		InitialSequenceName: "Course",
		InitialLessons:      []course.Lesson{{Name: "One", Content: "a"}, {Name: "Two", Content: "b"}},
	})
	next := press(m, runeKey('n'))
	if next.lessonIndex != 1 || next.inputKind != "" {
		t.Fatalf("doc n index %d kind %q", next.lessonIndex, next.inputKind)
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

func TestNewPartAndSectionFollowCursor(t *testing.T) {
	persistCourse = func(slug, name, body string) error { return nil }
	persistNewCourse = state.CreateCourse
	removeCourse = state.DeleteStoredCourse
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
	m.screen = screenParts
	m.courseIndex = 0
	m.selected = 0
	made := press(m, runeKey('n'), runeKey('D'), runeKey('e'), runeKey('l'), runeKey('t'), runeKey('a'), tea.KeyMsg{Type: tea.KeyEnter})
	if made.status != "created" || made.selected != 1 {
		t.Fatalf("status %q cursor %d", made.status, made.selected)
	}
	parts := made.cfg.Courses[0].Parts
	if parts[1].Name != "Part II: Delta" || parts[2].Name != "Part III: Two" {
		t.Fatalf("parts = %q %q %q", parts[0].Name, parts[1].Name, parts[2].Name)
	}

	m.screen = screenSections
	m.partIndex = 0
	m.selected = 0
	// Part I has one section, so grow it first, then insert between the two.
	grown := press(m, runeKey('n'), runeKey('B'), tea.KeyMsg{Type: tea.KeyEnter})
	if grown.selected != 1 || grown.cfg.Courses[0].Parts[0].Sections[1].Name != "Section 2: B" {
		t.Fatalf("grow status %q cursor %d names %q", grown.status, grown.selected, sectionNames(grown))
	}
	grown.selected = 0
	mid := press(grown, runeKey('n'), runeKey('Z'), tea.KeyMsg{Type: tea.KeyEnter})
	if mid.selected != 1 {
		t.Fatalf("cursor = %d", mid.selected)
	}
	got := sectionNames(mid)
	want := []string{"Section 1: A", "Section 2: Z", "Section 3: B"}
	if strings.Join(got, " | ") != strings.Join(want, " | ") {
		t.Fatalf("sections = %v", got)
	}
}

func sectionNames(m model) []string {
	sections := m.cfg.Courses[0].Parts[0].Sections
	out := make([]string, len(sections))
	for i, section := range sections {
		out[i] = section.Name
	}
	return out
}
