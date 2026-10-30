package app

import (
	"testing"

	"github.com/ryangerardwilson/c3po/nlp/internal/course"
)

func TestRestorePlaceUsesTheCourseSlug(t *testing.T) {
	lesson := course.Lesson{Name: "Start", Content: "    one"}
	m := newModel(Config{Courses: []course.Course{
		{
			Name: "Alpha", SourceFile: "alpha",
			Parts: []course.Part{{
				Name: "Part I: Alpha",
				Sections: []course.Section{{
					Name:    "Section 1: A",
					Lessons: []course.Lesson{lesson},
				}},
			}},
		},
		{
			Name: "Beta", SourceFile: "beta",
			Parts: []course.Part{{
				Name: "Main",
				Sections: []course.Section{{
					Name:    "Main",
					Lessons: []course.Lesson{lesson},
				}},
			}},
		},
	}})

	m.restore("beta", "courses 0")
	if m.screen != screenCourses || m.selected != 1 || m.courseIndex != 1 {
		t.Fatalf("course place screen=%d selected=%d course=%d", m.screen, m.selected, m.courseIndex)
	}

	m.restore("alpha", "doc 9 0 0 0")
	if m.screen != screenDoc || m.courseIndex != 0 || m.lessonIndex != 0 {
		t.Fatalf("doc place screen=%d course=%d lesson=%d", m.screen, m.courseIndex, m.lessonIndex)
	}
	if m.sequenceName != "Alpha: Part I: Alpha: Section 1: A" {
		t.Fatalf("sequence %q", m.sequenceName)
	}

	m.restore("alpha", "type 0 0 0 0")
	if m.screen != screenType || m.courseIndex != 0 {
		t.Fatalf("type place screen=%d course=%d", m.screen, m.courseIndex)
	}
}
