package app

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ryangerardwilson/c3po/nlp/internal/course"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestViewsFillTerminalHeight(t *testing.T) {
	m := testModel()
	m.width = 40
	m.height = 12

	for name, configure := range map[string]func(*model){
		"list": func(m *model) {
			m.screen = screenCourses
		},
		"doc": func(m *model) {
			m.screen = screenDoc
		},
		"type": func(m *model) {
			m.startTyping()
		},
		"help": func(m *model) {
			m.before = screenDoc
			m.screen = screenHelp
		},
	} {
		t.Run(name, func(t *testing.T) {
			current := m
			configure(&current)
			if got := renderedLineCount(current.View()); got != current.height {
				t.Fatalf("View() line count = %d, want %d\n%s", got, current.height, current.View())
			}
		})
	}
}

func TestTypeModeRendersTypedTextOverTargetLine(t *testing.T) {
	m := testModel()
	m.width = 40
	m.height = 12
	m.startTyping()
	m.typed = "abc"
	m.cursorShown = false

	view := stripANSI(m.View())
	if strings.Contains(view, "Typed:") {
		t.Fatalf("type mode should not render typed text below the target:\n%s", view)
	}
	if !strings.Contains(view, "abcdef") {
		t.Fatalf("typed overlay should preserve the target line with typed prefix over it:\n%s", view)
	}
}

func TestTypeModeRendersBlockCursorOnActiveLine(t *testing.T) {
	m := testModelWithContent("next\nsecond")
	m.width = 40
	m.height = 12
	m.startTyping()
	m.cursorShown = true

	view := stripANSI(m.View())
	if !strings.Contains(view, "█ext") {
		t.Fatalf("type mode should render a block cursor over the active character:\n%s", view)
	}
	if strings.Contains(view, "█econd") {
		t.Fatalf("type mode should render the cursor on the active line only:\n%s", view)
	}
}

func TestTypeModeHidesBlockCursorBetweenBlinks(t *testing.T) {
	m := testModelWithContent("next")
	m.width = 40
	m.height = 12
	m.startTyping()
	m.cursorShown = false

	view := stripANSI(m.View())
	if strings.Contains(view, "█") {
		t.Fatalf("hidden cursor frame should not render a block cursor:\n%s", view)
	}
	if !strings.Contains(view, "next") {
		t.Fatalf("hidden cursor frame should render the target text:\n%s", view)
	}
}

func TestTypeModeAllowsTypingNBeforeCompletion(t *testing.T) {
	m := testModelWithContent("next")
	m.startTyping()
	m.cursorShown = false

	updated, _ := m.updateTypeKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(model)

	if got := m.currentTypeInput(); got != "n" {
		t.Fatalf("typed input = %q, want n", got)
	}
	if m.typeDone {
		t.Fatalf("typing n should not complete or advance the lesson before the target is complete")
	}
	if !m.cursorShown {
		t.Fatalf("typing should make the block cursor visible immediately")
	}
}

func TestTypeModeKeepsLessonLinesVisible(t *testing.T) {
	m := testModel()
	m.width = 40
	m.height = 12
	m.startTyping()
	m.cursorShown = false

	view := stripANSI(m.View())
	for _, want := range []string{"abcdef", "second line", "third line"} {
		if !strings.Contains(view, want) {
			t.Fatalf("type mode should keep surrounding lesson lines visible; missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Line:") {
		t.Fatalf("type mode should render the lesson viewport, not a single line prompt:\n%s", view)
	}
}

func TestTypeModeScrollsAroundCurrentLine(t *testing.T) {
	m := testModelWithContent(strings.Join([]string{
		"line 01",
		"line 02",
		"line 03",
		"line 04",
		"line 05",
		"line 06",
		"line 07",
		"line 08",
		"line 09",
		"line 10",
		"line 11",
		"line 12",
	}, "\n"))
	m.width = 40
	m.height = 10
	m.startTyping()
	m.typeLine = 8
	m.cursorShown = false

	view := stripANSI(m.View())
	for _, want := range []string{"line 08", "line 09", "line 12"} {
		if !strings.Contains(view, want) {
			t.Fatalf("type mode should keep the current line and nearby lines visible; missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "line 01") {
		t.Fatalf("type mode should scroll long lessons instead of pinning the first line:\n%s", view)
	}
}

func TestStartTypingKeepsBlankAndSkipLines(t *testing.T) {
	m := testModelWithContent("first\n\n//! run this outside typing\nlast\n")
	m.startTyping()

	if got, want := len(m.typeLines), 4; got != want {
		t.Fatalf("typeLines length = %d, want %d: %#v", got, want, m.typeLines)
	}
	if !m.typeSkip[2] {
		t.Fatalf("skip line was not marked: %#v", m.typeSkip)
	}
	_, total := m.typeStats()
	if total != len("first")+len("last") {
		t.Fatalf("typeStats total = %d, want non-skip characters only", total)
	}
}

func testModel() model {
	return testModelWithContent("abcdef\nsecond line\nthird line")
}

func testModelWithContent(content string) model {
	return newModel(Config{
		Courses: []course.Course{{
			Name: "Course",
			Parts: []course.Part{{
				Name: "Main",
				Sections: []course.Section{{
					Name: "Main",
					Lessons: []course.Lesson{{
						Name:    "Lesson",
						Content: content,
					}},
				}},
			}},
		}},
		InitialSequenceName: "Course",
		InitialLessons: []course.Lesson{{
			Name:    "Lesson",
			Content: content,
		}},
	})
}

func renderedLineCount(value string) int {
	return strings.Count(value, "\n") + 1
}

func stripANSI(value string) string {
	return ansiRE.ReplaceAllString(value, "")
}
