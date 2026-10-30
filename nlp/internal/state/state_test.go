package state

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/nlp/internal/course"
)

func TestLoadCoursesSeedsEmbeddedAndImportsLegacyOnce(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "sqlx")
	t.Setenv("HOME", home)
	t.Setenv("SQLX_ROOT", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))

	coursePath := filepath.Join(home, "custom.md")
	body := "# Custom\n\n## Lesson\n\n    echo ok\n"
	if err := os.WriteFile(coursePath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(home, "config", "rt", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "{\n  \"courses\": [{\"name\": \"Custom\", \"local_path\": \"" + coursePath + "\"}]\n}\n"
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	courses, err := LoadCourses("0.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !hasCourse(courses, "Custom") || !hasCourse(courses, "Unix") || !hasCourse(courses, "Oil") {
		t.Fatalf("missing seeded courses: %v", names(courses))
	}

	db := filepath.Join(root, "data", "nlp.sqlite")
	cmd := exec.Command("/usr/bin/sqlite3", db, "DELETE FROM course WHERE slug = 'custom';")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("delete custom: %v\n%s", err, out)
	}
	courses, err = LoadCourses("0.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if hasCourse(courses, "Custom") {
		t.Fatal("deleted custom course was read from config again")
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("legacy config was removed: %v", err)
	}
	if _, err := os.Stat(coursePath); err != nil {
		t.Fatalf("legacy course file was removed: %v", err)
	}
}

func TestVersionRefreshUpdatesBundledCourseOnly(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "sqlx")
	t.Setenv("HOME", home)
	t.Setenv("SQLX_ROOT", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))

	if _, err := LoadCourses("0.0.0"); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(root, "data", "nlp.sqlite")
	edit := "UPDATE course SET body = '# Unix\n\n## Lesson\n\n    changed\n' WHERE slug = 'unix';"
	cmd := exec.Command("/usr/bin/sqlite3", db, edit)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("edit unix: %v\n%s", err, out)
	}
	courses, err := LoadCourses("0.0.0")
	if err != nil {
		t.Fatal(err)
	}
	unix := findCourse(t, courses, "Unix")
	if !containsLesson(unix, "changed") {
		t.Fatal("same version overwrote a bundled course edit")
	}
	courses, err = LoadCourses("0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	unix = findCourse(t, courses, "Unix")
	if containsLesson(unix, "changed") {
		t.Fatal("version refresh kept the edited bundled body")
	}
	if !containsLesson(unix, "Option A") {
		t.Fatal("version refresh did not restore the embedded unix course")
	}
}

func hasCourse(courses []course.Course, name string) bool {
	for _, item := range courses {
		if item.Name == name {
			return true
		}
	}
	return false
}

func names(courses []course.Course) []string {
	out := make([]string, 0, len(courses))
	for _, item := range courses {
		out = append(out, item.Name)
	}
	return out
}

func findCourse(t *testing.T, courses []course.Course, name string) course.Course {
	t.Helper()
	for _, item := range courses {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("course %s not found in %v", name, names(courses))
	return course.Course{}
}

func containsLesson(item course.Course, text string) bool {
	for _, part := range item.Parts {
		for _, section := range part.Sections {
			for _, lesson := range section.Lessons {
				if strings.Contains(lesson.Content, text) {
					return true
				}
			}
		}
	}
	return false
}

func TestSaveCourseSurvivesVersionRefresh(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "sqlx")
	t.Setenv("HOME", home)
	t.Setenv("SQLX_ROOT", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))

	courses, err := LoadCourses("0.0.0")
	if err != nil {
		t.Fatal(err)
	}
	unix := findCourse(t, courses, "Unix")
	if !strings.Contains(unix.Body, "Option A") {
		t.Fatal("unix body was not kept on the course")
	}
	body := strings.Replace(unix.Body, "Option A", "Option Saved", 1)
	if err := SaveCourse("unix", "Unix", body); err != nil {
		t.Fatal(err)
	}
	courses, err = LoadCourses("0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	unix = findCourse(t, courses, "Unix")
	if !containsLesson(unix, "Option Saved") {
		t.Fatal("version refresh restored a menu-saved course")
	}
}

func TestDeletedBundledCourseStaysRetired(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "sqlx")
	t.Setenv("HOME", home)
	t.Setenv("SQLX_ROOT", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))

	if _, err := LoadCourses("0.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteStoredCourse("unix"); err != nil {
		t.Fatal(err)
	}
	courses, err := LoadCourses("0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if hasCourse(courses, "Unix") {
		t.Fatal("retired bundled course returned")
	}
	if _, err := CreateCourse("Unix", course.NewCourseBody("Unix")); err != nil {
		t.Fatal(err)
	}
	courses, err = LoadCourses("0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	if !hasCourse(courses, "Unix") {
		t.Fatal("an explicitly created course was dropped")
	}
}

func TestCreateCourseRejectsDuplicateSlug(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "sqlx")
	t.Setenv("HOME", home)
	t.Setenv("SQLX_ROOT", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))

	if _, err := LoadCourses("0.0.0"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateCourse("Unix", course.NewCourseBody("Unix")); err == nil || !strings.Contains(err.Error(), "course already exists") {
		t.Fatalf("duplicate slug err = %v", err)
	}
	slug, err := CreateCourse("Widgets", course.NewCourseBody("Widgets"))
	if err != nil {
		t.Fatal(err)
	}
	if slug != "widgets" {
		t.Fatalf("slug = %s", slug)
	}
	courses, err := LoadCourses("9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	if !hasCourse(courses, "Widgets") {
		t.Fatal("created course did not survive a version refresh")
	}
}
