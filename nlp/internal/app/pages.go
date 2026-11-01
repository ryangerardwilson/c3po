package app

import (
	"strings"

	"github.com/ryangerardwilson/c3po/nlp/internal/course"
)

// page is one practice session. The list is these pages, and opening one
// starts touch typing that page.
type page struct {
	name         string
	courseIndex  int
	partIndex    int
	sectionIndex int
	lessonIndex  int
}

func pageTitle(lessonName string) string {
	name := strings.TrimSpace(lessonName)
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, "lesson ") || strings.HasPrefix(lower, "lession ") {
		if colon := strings.Index(name, ":"); colon >= 0 {
			rest := strings.TrimSpace(name[colon+1:])
			if rest != "" {
				return rest
			}
		}
	}
	return name
}

func lessonCount(item course.Course) int {
	n := 0
	for _, part := range item.Parts {
		for _, section := range part.Sections {
			n += len(section.Lessons)
		}
	}
	return n
}

func (m *model) rebuildPages() {
	type raw struct {
		page
		base   string
		course string
	}
	var items []raw
	counts := map[string]int{}
	same := map[string]int{}
	for ci, item := range m.cfg.Courses {
		for pi, part := range item.Parts {
			for si, section := range part.Sections {
				for li, lesson := range section.Lessons {
					base := pageTitle(lesson.Name)
					counts[base]++
					same[base+"\x00"+item.Name]++
					items = append(items, raw{
						page: page{
							courseIndex:  ci,
							partIndex:    pi,
							sectionIndex: si,
							lessonIndex:  li,
						},
						base:   base,
						course: item.Name,
					})
				}
			}
		}
	}
	pages := make([]page, 0, len(items))
	for _, item := range items {
		name := item.base
		if counts[item.base] > 1 {
			name = item.base + " (" + item.course + ")"
		}
		if same[item.base+"\x00"+item.course] > 1 {
			lesson := m.cfg.Courses[item.courseIndex].Parts[item.partIndex].Sections[item.sectionIndex].Lessons[item.lessonIndex]
			name = strings.TrimSpace(lesson.Name) + " (" + item.course + ")"
		}
		item.page.name = name
		pages = append(pages, item.page)
	}
	for i := 1; i < len(pages); i++ {
		item := pages[i]
		j := i
		for j > 0 && strings.ToLower(pages[j-1].name) > strings.ToLower(item.name) {
			pages[j] = pages[j-1]
			j--
		}
		pages[j] = item
	}
	m.pages = pages
}

func (m *model) openPage(index int) {
	if index < 0 || index >= len(m.pages) {
		return
	}
	p := m.pages[index]
	if p.courseIndex < 0 || p.courseIndex >= len(m.cfg.Courses) {
		return
	}
	item := m.cfg.Courses[p.courseIndex]
	if p.partIndex < 0 || p.partIndex >= len(item.Parts) {
		return
	}
	part := item.Parts[p.partIndex]
	if p.sectionIndex < 0 || p.sectionIndex >= len(part.Sections) {
		return
	}
	section := part.Sections[p.sectionIndex]
	if p.lessonIndex < 0 || p.lessonIndex >= len(section.Lessons) {
		return
	}
	m.selected = index
	m.courseIndex = p.courseIndex
	m.partIndex = p.partIndex
	m.sectionIndex = p.sectionIndex
	m.lessonIndex = 0
	m.sequenceName = p.name
	m.lessons = []course.Lesson{section.Lessons[p.lessonIndex]}
	m.scroll = 0
	m.startTyping()
}

func (m *model) selectStoredPage(slug string, partIndex, sectionIndex, lessonIndex int) {
	for i, p := range m.pages {
		if p.courseIndex < 0 || p.courseIndex >= len(m.cfg.Courses) {
			continue
		}
		if m.cfg.Courses[p.courseIndex].SourceFile != slug {
			continue
		}
		if p.partIndex == partIndex && p.sectionIndex == sectionIndex && p.lessonIndex == lessonIndex {
			m.selected = i
			m.courseIndex = p.courseIndex
			return
		}
	}
	m.selected = m.pageIndex(slug, 0)
	if m.selected >= 0 && m.selected < len(m.pages) {
		m.courseIndex = m.pages[m.selected].courseIndex
	}
}

func (m *model) pageIndex(slug string, index int) int {
	if len(m.pages) == 0 {
		return 0
	}
	if index >= 0 && index < len(m.pages) {
		p := m.pages[index]
		if slug == "" || (p.courseIndex < len(m.cfg.Courses) && m.cfg.Courses[p.courseIndex].SourceFile == slug) {
			return index
		}
	}
	if slug != "" {
		for i, p := range m.pages {
			if p.courseIndex < len(m.cfg.Courses) && m.cfg.Courses[p.courseIndex].SourceFile == slug {
				return i
			}
		}
	}
	return clamp(index, 0, len(m.pages)-1)
}

func (m *model) pageByLesson(courseIndex, partIndex, sectionIndex, lessonIndex int) int {
	for i, p := range m.pages {
		if p.courseIndex == courseIndex && p.partIndex == partIndex && p.sectionIndex == sectionIndex && p.lessonIndex == lessonIndex {
			return i
		}
	}
	for i, p := range m.pages {
		if p.courseIndex == courseIndex {
			return i
		}
	}
	return 0
}

func courseIndexBySlug(courses []course.Course, slug string) int {
	for i, item := range courses {
		if item.SourceFile == slug {
			return i
		}
	}
	return -1
}
