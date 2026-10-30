package course

import (
	"testing"
	"testing/fstest"
)

func TestParseEmbeddedCourses(t *testing.T) {
	courses, err := ParseEmbeddedCourses()
	if err != nil {
		t.Fatal(err)
	}
	if len(courses) == 0 {
		t.Fatal("expected embedded courses")
	}
	if courses[0].Name == "" || len(courses[0].Parts) == 0 {
		t.Fatalf("invalid parsed course: %#v", courses[0])
	}
	names := map[string]bool{}
	for _, item := range courses {
		names[item.Name] = true
	}
	if !names["Unix"] || !names["Oil"] {
		t.Fatalf("embedded courses = %#v", names)
	}
}

func TestParseFlatHierarchy(t *testing.T) {
	files := fstest.MapFS{
		"demo.md": {Data: []byte("# Demo\n\n## Lesson A\n\n    alpha\n\n## Lesson B\n\n    beta\n")},
	}
	courses, err := ParseCourses(files, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(courses) != 1 {
		t.Fatalf("courses = %d", len(courses))
	}
	lessons := courses[0].Parts[0].Sections[0].Lessons
	if len(lessons) != 2 || lessons[0].Name != "Lesson A" || lessons[0].Content[:5] != "alpha" {
		t.Fatalf("unexpected lessons: %#v", lessons)
	}
}
