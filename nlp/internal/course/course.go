package course

import (
	"bufio"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	root "github.com/ryangerardwilson/c3po/nlp/bundle"
)

type Lesson struct {
	Name    string
	Content string
}

type Section struct {
	Name    string
	Lessons []Lesson
}

type Part struct {
	Name     string
	Sections []Section
}

type Course struct {
	Name       string
	Parts      []Part
	SourceFile string
	Body       string
}

func ParseEmbeddedCourses() ([]Course, error) {
	return ParseCourses(root.CourseFiles, "courses")
}

func ParseCourses(files fs.FS, dir string) ([]Course, error) {
	entries, err := fs.ReadDir(files, dir)
	if err != nil {
		return nil, err
	}
	courses := make([]Course, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.ToSlash(filepath.Join(dir, entry.Name()))
		body, err := fs.ReadFile(files, path)
		if err != nil {
			continue
		}
		parsed, ok := ParseMarkdown(path, string(body))
		if ok {
			courses = append(courses, parsed)
		}
	}
	sort.Slice(courses, func(i, j int) bool {
		return strings.ToLower(courses[i].Name) < strings.ToLower(courses[j].Name)
	})
	return courses, nil
}

func ParseMarkdown(sourceFile, body string) (Course, bool) {
	lines := scanLines(body)
	hasSections := false
	hasParts := false
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "#### ") {
			hasSections = true
		}
		if strings.HasPrefix(trimmed, "### ") {
			hasParts = true
		}
	}
	var parsed Course
	var ok bool
	switch {
	case hasSections:
		parsed, ok = parseFullHierarchy(sourceFile, lines)
	case hasParts:
		parsed, ok = parseMidHierarchy(sourceFile, lines)
	default:
		parsed, ok = parseFlatHierarchy(sourceFile, lines)
	}
	if ok {
		parsed.Body = body
	}
	return parsed, ok
}

func parseFullHierarchy(sourceFile string, lines []string) (Course, bool) {
	var courseName string
	var parts []Part
	var currentPart *Part
	var currentSection *Section
	var lessonName string
	var content []string
	inCode := false

	flushLesson := func() {
		if lessonName == "" || len(content) == 0 || currentSection == nil {
			return
		}
		currentSection.Lessons = append(currentSection.Lessons, Lesson{Name: lessonName, Content: finishContent(content)})
		lessonName = ""
		content = nil
	}
	flushSection := func() {
		flushLesson()
		if currentSection != nil && currentPart != nil {
			currentPart.Sections = append(currentPart.Sections, *currentSection)
			currentSection = nil
		}
	}
	flushPart := func() {
		flushSection()
		if currentPart != nil {
			parts = append(parts, *currentPart)
			currentPart = nil
		}
	}

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "# "):
			if courseName != "" {
				return Course{}, false
			}
			courseName = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		case strings.HasPrefix(line, "## "):
			flushPart()
			currentPart = &Part{Name: strings.TrimSpace(strings.TrimPrefix(line, "## "))}
			inCode = false
		case strings.HasPrefix(line, "### "):
			flushSection()
			if currentPart == nil {
				return Course{}, false
			}
			currentSection = &Section{Name: strings.TrimSpace(strings.TrimPrefix(line, "### "))}
			inCode = false
		case strings.HasPrefix(line, "#### "):
			flushLesson()
			if currentSection == nil {
				return Course{}, false
			}
			lessonName = strings.TrimSpace(strings.TrimPrefix(line, "#### "))
			inCode = false
		default:
			content, inCode = appendLessonContent(line, lessonName, content, inCode)
		}
	}
	flushPart()
	if courseName == "" || len(parts) == 0 {
		return Course{}, false
	}
	return Course{Name: courseName, Parts: parts, SourceFile: sourceFile}, true
}

func parseMidHierarchy(sourceFile string, lines []string) (Course, bool) {
	var courseName string
	var parts []Part
	var currentPart *Part
	var lessonName string
	var content []string
	inCode := false

	flushLesson := func() {
		if lessonName == "" || len(content) == 0 || currentPart == nil {
			return
		}
		currentPart.Sections[0].Lessons = append(currentPart.Sections[0].Lessons, Lesson{Name: lessonName, Content: finishContent(content)})
		lessonName = ""
		content = nil
	}
	flushPart := func() {
		flushLesson()
		if currentPart != nil {
			parts = append(parts, *currentPart)
			currentPart = nil
		}
	}

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "# "):
			if courseName != "" {
				return Course{}, false
			}
			courseName = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		case strings.HasPrefix(line, "## "):
			flushPart()
			currentPart = &Part{Name: strings.TrimSpace(strings.TrimPrefix(line, "## ")), Sections: []Section{{Name: "Main"}}}
			inCode = false
		case strings.HasPrefix(line, "### "):
			flushLesson()
			if currentPart == nil {
				return Course{}, false
			}
			lessonName = strings.TrimSpace(strings.TrimPrefix(line, "### "))
			inCode = false
		default:
			content, inCode = appendLessonContent(line, lessonName, content, inCode)
		}
	}
	flushPart()
	if courseName == "" || len(parts) == 0 {
		return Course{}, false
	}
	return Course{Name: courseName, Parts: parts, SourceFile: sourceFile}, true
}

func parseFlatHierarchy(sourceFile string, lines []string) (Course, bool) {
	var courseName string
	var lessons []Lesson
	var lessonName string
	var content []string
	inCode := false

	flushLesson := func() {
		if lessonName == "" || len(content) == 0 {
			return
		}
		lessons = append(lessons, Lesson{Name: lessonName, Content: finishContent(content)})
		lessonName = ""
		content = nil
	}

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "# "):
			if courseName != "" {
				return Course{}, false
			}
			courseName = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		case strings.HasPrefix(line, "## "):
			flushLesson()
			lessonName = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			inCode = false
		default:
			content, inCode = appendLessonContent(line, lessonName, content, inCode)
		}
	}
	flushLesson()
	if courseName == "" || len(lessons) == 0 {
		return Course{}, false
	}
	return Course{
		Name:       courseName,
		Parts:      []Part{{Name: "Main", Sections: []Section{{Name: "Main", Lessons: lessons}}}},
		SourceFile: sourceFile,
	}, true
}

func appendLessonContent(line, lessonName string, content []string, inCode bool) ([]string, bool) {
	if lessonName == "" {
		return content, inCode
	}
	if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
		if strings.HasPrefix(line, "    ") {
			content = append(content, strings.TrimRight(line[4:], " \t"))
		} else {
			content = append(content, strings.TrimRight(line[1:], " \t"))
		}
		return content, true
	}
	if inCode && strings.TrimSpace(line) == "" {
		content = append(content, "")
		return content, true
	}
	if inCode && strings.TrimSpace(line) != "" {
		return content, false
	}
	return content, inCode
}

func finishContent(lines []string) string {
	return strings.Join(lines, "\n") + "\n\n\n\n\n\n\n"
}

func scanLines(body string) []string {
	scanner := bufio.NewScanner(strings.NewReader(body))
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func FlattenLessons(course Course, partName, sectionName string) (string, []Lesson, bool) {
	for _, part := range course.Parts {
		if partName != "" && part.Name != partName {
			continue
		}
		if sectionName != "" {
			for _, section := range part.Sections {
				if section.Name == sectionName {
					return course.Name + ": " + part.Name + ": " + section.Name, section.Lessons, true
				}
			}
			continue
		}
		lessons := []Lesson{}
		for _, section := range part.Sections {
			lessons = append(lessons, section.Lessons...)
		}
		if len(lessons) > 0 {
			if partName != "" {
				return course.Name + ": " + part.Name, lessons, true
			}
			return course.Name, lessons, true
		}
	}
	return "", nil, false
}
