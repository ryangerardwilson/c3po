package course

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

var partNumber = map[string]int{}

func init() {
	for n := 1; n <= 40; n++ {
		partNumber[roman(n)] = n
	}
}

// NewCourseBody is one practice page.
func NewCourseBody(name string) string {
	name = cleanName(name)
	if name == "" {
		name = "Session"
	}
	return "# " + name + "\n\n## " + name + "\n\n    ...\n"
}

// RenameCourse replaces the course heading.
func RenameCourse(body, name string) (string, error) {
	name = cleanName(name)
	if name == "" {
		return "", fmt.Errorf("missing name")
	}
	lines, err := headingLines(body)
	if err != nil {
		return "", err
	}
	found := false
	for i, line := range lines {
		level, _, ok := headingTitle(line)
		if ok && level == 1 {
			lines[i] = headingLine(1, name)
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("course heading not found")
	}
	return joinLines(lines), nil
}

// RenamePart replaces the full part menu label.
func RenamePart(body string, index int, name string) (string, error) {
	return renameAt(body, 2, index, -1, name)
}

// RenameSection replaces the full section menu label.
func RenameSection(body string, partIndex, sectionIndex int, name string) (string, error) {
	return renameAt(body, 3, partIndex, sectionIndex, name)
}

// DeletePart removes a part and closes Part roman numbering.
func DeletePart(body string, index int) (string, error) {
	lines, err := headingLines(body)
	if err != nil {
		return "", err
	}
	parts := partSpans(lines)
	if index < 0 || index >= len(parts) {
		return "", fmt.Errorf("part not found")
	}
	if len(parts) < 2 {
		return "", fmt.Errorf("keep at least one part")
	}
	span := parts[index]
	lines = append(lines[:span.start], lines[span.end:]...)
	renumberParts(lines)
	return joinLines(lines), nil
}

// DeleteSection removes a section and closes Section numbering in that part.
func DeleteSection(body string, partIndex, sectionIndex int) (string, error) {
	lines, err := headingLines(body)
	if err != nil {
		return "", err
	}
	parts := partSpans(lines)
	if partIndex < 0 || partIndex >= len(parts) {
		return "", fmt.Errorf("part not found")
	}
	sections := sectionSpans(lines, parts[partIndex])
	if sectionIndex < 0 || sectionIndex >= len(sections) {
		return "", fmt.Errorf("section not found")
	}
	if len(sections) < 2 {
		return "", fmt.Errorf("keep at least one section")
	}
	span := sections[sectionIndex]
	lines = append(lines[:span.start], lines[span.end:]...)
	parts = partSpans(lines)
	if partIndex >= len(parts) {
		return "", fmt.Errorf("part not found")
	}
	renumberSections(lines, parts[partIndex])
	return joinLines(lines), nil
}

// InsertPart inserts a part after the part at after. withSections chooses a sectioned part or a lesson part.
func InsertPart(body, title string, after int, withSections bool) (string, error) {
	title = subtitle(title)
	if title == "" {
		return "", fmt.Errorf("missing name")
	}
	lines, err := headingLines(body)
	if err != nil {
		return "", err
	}
	parts := partSpans(lines)
	at := len(lines)
	if len(parts) > 0 {
		if after < 0 || after >= len(parts) {
			return "", fmt.Errorf("part not found")
		}
		at = parts[after].end
	}
	titles := make([]string, len(parts))
	for i, span := range parts {
		_, titles[i], _ = headingTitle(lines[span.start])
	}
	block := partBlock(nextPartLabel(titles), title, withSections)
	lines = insertBlock(lines, at, block)
	renumberParts(lines)
	return joinLines(lines), nil
}

// InsertSection inserts a section after the section at after, using that part's numbering.
func InsertSection(body string, partIndex, after int, title string) (string, error) {
	title = subtitle(title)
	if title == "" {
		return "", fmt.Errorf("missing name")
	}
	lines, err := headingLines(body)
	if err != nil {
		return "", err
	}
	parts := partSpans(lines)
	if partIndex < 0 || partIndex >= len(parts) {
		return "", fmt.Errorf("part not found")
	}
	part := parts[partIndex]
	sections := sectionSpans(lines, part)
	at := part.end
	if len(sections) > 0 {
		if after < 0 || after >= len(sections) {
			return "", fmt.Errorf("section not found")
		}
		at = sections[after].end
	}
	titles := make([]string, len(sections))
	for i, span := range sections {
		_, titles[i], _ = headingTitle(lines[span.start])
	}
	lines = insertBlock(lines, at, sectionBlock(nextSectionLabel(titles), title))
	parts = partSpans(lines)
	if partIndex >= len(parts) {
		return "", fmt.Errorf("part not found")
	}
	renumberSections(lines, parts[partIndex])
	return joinLines(lines), nil
}

func insertBlock(lines []string, at int, block string) []string {
	insert := strings.Split(strings.TrimSuffix(block, "\n"), "\n")
	if at > 0 && strings.TrimSpace(lines[at-1]) != "" {
		insert = append([]string{""}, insert...)
	}
	if at < len(lines) && strings.TrimSpace(lines[at]) != "" {
		insert = append(insert, "")
	}
	out := make([]string, 0, len(lines)+len(insert))
	out = append(out, lines[:at]...)
	out = append(out, insert...)
	out = append(out, lines[at:]...)
	return out
}

func renameAt(body string, level, partIndex, sectionIndex int, name string) (string, error) {
	name = cleanName(name)
	if name == "" {
		return "", fmt.Errorf("missing name")
	}
	lines, err := headingLines(body)
	if err != nil {
		return "", err
	}
	line, err := menuLine(lines, level, partIndex, sectionIndex)
	if err != nil {
		return "", err
	}
	current, _, ok := headingTitle(lines[line])
	if !ok {
		return "", fmt.Errorf("heading not found")
	}
	lines[line] = headingLine(current, name)
	return joinLines(lines), nil
}

func menuLine(lines []string, level, partIndex, sectionIndex int) (int, error) {
	parts := partSpans(lines)
	if level == 2 {
		if partIndex < 0 || partIndex >= len(parts) {
			return 0, fmt.Errorf("part not found")
		}
		return parts[partIndex].start, nil
	}
	if partIndex < 0 || partIndex >= len(parts) {
		return 0, fmt.Errorf("part not found")
	}
	sections := sectionSpans(lines, parts[partIndex])
	if sectionIndex < 0 || sectionIndex >= len(sections) {
		return 0, fmt.Errorf("section not found")
	}
	return sections[sectionIndex].start, nil
}

type span struct {
	start int
	end   int
}

func partSpans(lines []string) []span {
	starts := []int{}
	for i, line := range lines {
		level, _, ok := headingTitle(line)
		if ok && level == 2 {
			starts = append(starts, i)
		}
	}
	return spansUntil(starts, len(lines), lines, 2)
}

func sectionSpans(lines []string, part span) []span {
	starts := []int{}
	for i := part.start + 1; i < part.end; i++ {
		level, _, ok := headingTitle(lines[i])
		if ok && level == 3 {
			starts = append(starts, i)
		}
	}
	limit := part.end
	return spansUntil(starts, limit, lines, 3)
}

func spansUntil(starts []int, limit int, lines []string, level int) []span {
	out := make([]span, len(starts))
	for i, start := range starts {
		end := limit
		for j := start + 1; j < limit; j++ {
			next, _, ok := headingTitle(lines[j])
			if ok && next > 0 && next <= level {
				end = j
				break
			}
		}
		out[i] = span{start: start, end: end}
	}
	return out
}

func renumberParts(lines []string) {
	n := 1
	for i, line := range lines {
		level, title, ok := headingTitle(line)
		if !ok || level != 2 {
			continue
		}
		rest, ok := partRest(title)
		if !ok {
			continue
		}
		lines[i] = headingLine(2, "Part "+roman(n)+": "+rest)
		n++
	}
}

func renumberSections(lines []string, part span) {
	number := 1
	letter := 1
	for i := part.start; i < part.end && i < len(lines); i++ {
		level, title, ok := headingTitle(lines[i])
		if !ok || level != 3 {
			continue
		}
		if rest, ok := sectionNumberRest(title); ok {
			lines[i] = headingLine(3, fmt.Sprintf("Section %d: %s", number, rest))
			number++
			continue
		}
		if rest, ok := sectionLetterRest(title); ok {
			if letter > 26 {
				continue
			}
			lines[i] = headingLine(3, "Section "+string(rune('A'+letter-1))+": "+rest)
			letter++
		}
	}
}

func nextPartLabel(titles []string) string {
	maxN := 0
	saw := false
	for _, title := range titles {
		token, _, ok := cutPart(title)
		if !ok {
			continue
		}
		n := partNumber[token]
		saw = true
		if n > maxN {
			maxN = n
		}
	}
	n := len(titles) + 1
	if saw {
		n = maxN + 1
	}
	return "Part " + roman(n)
}

func nextSectionLabel(titles []string) string {
	maxNum := 0
	maxLet := 0
	sawNum := false
	sawLet := false
	last := ""
	for _, title := range titles {
		if rest, ok := sectionNumberRest(title); ok {
			_ = rest
			n := sectionNumber(title)
			sawNum = true
			last = "num"
			if n > maxNum {
				maxNum = n
			}
			continue
		}
		if _, ok := sectionLetterRest(title); ok {
			n := sectionLetter(title)
			sawLet = true
			last = "let"
			if n > maxLet {
				maxLet = n
			}
		}
	}
	if last == "let" || (sawLet && !sawNum) {
		n := maxLet + 1
		if n < 1 {
			n = 1
		}
		if n > 26 {
			return fmt.Sprintf("Section %d", maxNum+1)
		}
		return "Section " + string(rune('A'+n-1))
	}
	n := len(titles) + 1
	if sawNum {
		n = maxNum + 1
	}
	return fmt.Sprintf("Section %d", n)
}

func partBlock(label, title string, withSections bool) string {
	if withSections {
		return "## " + label + ": " + title + "\n\n### Section 1: Start\n\n#### Lesson 1: Start\n\n    ...\n"
	}
	return "## " + label + ": " + title + "\n\n### Lesson 1: Start\n\n    ...\n"
}

func sectionBlock(label, title string) string {
	return "### " + label + ": " + title + "\n\n#### Lesson 1: Start\n\n    ...\n"
}

func partRest(title string) (string, bool) {
	_, rest, ok := cutPart(title)
	return rest, ok
}

func cutPart(title string) (string, string, bool) {
	if !strings.HasPrefix(title, "Part ") {
		return "", "", false
	}
	token, rest, ok := strings.Cut(strings.TrimPrefix(title, "Part "), ":")
	if !ok {
		return "", "", false
	}
	token = strings.TrimSpace(token)
	if _, known := partNumber[token]; !known {
		return "", "", false
	}
	return token, strings.TrimSpace(rest), true
}

func sectionNumberRest(title string) (string, bool) {
	if !strings.HasPrefix(title, "Section ") {
		return "", false
	}
	token, rest, ok := strings.Cut(strings.TrimPrefix(title, "Section "), ":")
	if !ok {
		return "", false
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}
	for _, r := range token {
		if !unicode.IsDigit(r) {
			return "", false
		}
	}
	return strings.TrimSpace(rest), true
}

func sectionLetterRest(title string) (string, bool) {
	if !strings.HasPrefix(title, "Section ") {
		return "", false
	}
	token, rest, ok := strings.Cut(strings.TrimPrefix(title, "Section "), ":")
	if !ok {
		return "", false
	}
	token = strings.TrimSpace(token)
	if len(token) != 1 || token[0] < 'A' || token[0] > 'Z' {
		return "", false
	}
	return strings.TrimSpace(rest), true
}

func sectionNumber(title string) int {
	token, _, _ := strings.Cut(strings.TrimPrefix(title, "Section "), ":")
	n, _ := strconv.Atoi(strings.TrimSpace(token))
	return n
}

func sectionLetter(title string) int {
	token, _, _ := strings.Cut(strings.TrimPrefix(title, "Section "), ":")
	token = strings.TrimSpace(token)
	if token == "" {
		return 0
	}
	return int(token[0] - 'A' + 1)
}

func subtitle(title string) string {
	title = cleanName(title)
	if _, rest, ok := cutPart(title); ok {
		return rest
	}
	if rest, ok := sectionNumberRest(title); ok {
		return rest
	}
	if rest, ok := sectionLetterRest(title); ok {
		return rest
	}
	return title
}

func cleanName(name string) string {
	name = strings.TrimSpace(name)
	if strings.ContainsAny(name, "\r\n") {
		return ""
	}
	return name
}

func headingLines(body string) ([]string, error) {
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("course has no stored text")
	}
	return strings.Split(strings.TrimSuffix(body, "\n"), "\n"), nil
}

func joinLines(lines []string) string {
	return strings.Join(lines, "\n") + "\n"
}

func headingTitle(line string) (int, string, bool) {
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i == 0 || i >= len(line) || line[i] != ' ' {
		return 0, "", false
	}
	return i, strings.TrimSpace(line[i+1:]), true
}

func headingLine(level int, title string) string {
	return strings.Repeat("#", level) + " " + title
}

func roman(n int) string {
	if n < 1 {
		n = 1
	}
	values := []int{40, 10, 9, 5, 4, 1}
	symbols := []string{"XL", "X", "IX", "V", "IV", "I"}
	var b strings.Builder
	for i, value := range values {
		for n >= value {
			b.WriteString(symbols[i])
			n -= value
		}
	}
	return b.String()
}

// RenameLesson replaces one lesson heading with the typed name.
func RenameLesson(body string, partIndex, sectionIndex, lessonIndex int, name string) (string, error) {
	name = cleanName(name)
	if name == "" {
		return "", fmt.Errorf("missing name")
	}
	lines, err := headingLines(body)
	if err != nil {
		return "", err
	}
	at, level, ok := findLesson(lines, partIndex, sectionIndex, lessonIndex)
	if !ok {
		return "", fmt.Errorf("session not found")
	}
	lines[at] = headingLine(level, name)
	return joinLines(lines), nil
}

// DeleteLesson removes one lesson page. The last page of a course stays.
func DeleteLesson(body string, partIndex, sectionIndex, lessonIndex int) (string, error) {
	lines, err := headingLines(body)
	if err != nil {
		return "", err
	}
	if countLessonHeadings(lines) < 2 {
		return "", fmt.Errorf("keep at least one session")
	}
	at, level, ok := findLesson(lines, partIndex, sectionIndex, lessonIndex)
	if !ok {
		return "", fmt.Errorf("session not found")
	}
	end := len(lines)
	for i := at + 1; i < len(lines); i++ {
		next, _, isHeading := headingTitle(lines[i])
		if isHeading && next <= level {
			end = i
			break
		}
	}
	lines = append(lines[:at], lines[end:]...)
	return joinLines(lines), nil
}

func countLessonHeadings(lines []string) int {
	level := lessonHeadingLevel(lines)
	n := 0
	for _, line := range lines {
		got, _, ok := headingTitle(line)
		if ok && got == level {
			n++
		}
	}
	return n
}

func lessonHeadingLevel(lines []string) int {
	has4 := false
	has3 := false
	for _, line := range lines {
		level, _, ok := headingTitle(line)
		if !ok {
			continue
		}
		if level >= 4 {
			has4 = true
		}
		if level == 3 {
			has3 = true
		}
	}
	if has4 {
		return 4
	}
	if has3 {
		return 3
	}
	return 2
}

func findLesson(lines []string, partIndex, sectionIndex, lessonIndex int) (int, int, bool) {
	level := lessonHeadingLevel(lines)
	part := -1
	section := -1
	lesson := -1
	for i, line := range lines {
		n, _, ok := headingTitle(line)
		if !ok {
			continue
		}
		switch {
		case level == 4 && n == 2:
			part++
			section = -1
			lesson = -1
		case level == 4 && n == 3:
			section++
			lesson = -1
		case level == 4 && n == 4:
			lesson++
			if part == partIndex && section == sectionIndex && lesson == lessonIndex {
				return i, 4, true
			}
		case level == 3 && n == 2:
			part++
			section = 0
			lesson = -1
		case level == 3 && n == 3:
			lesson++
			if part == partIndex && sectionIndex == 0 && lesson == lessonIndex {
				return i, 3, true
			}
		case level == 2 && n == 2:
			lesson++
			if partIndex == 0 && sectionIndex == 0 && lesson == lessonIndex {
				return i, 2, true
			}
		}
	}
	return 0, 0, false
}

// PageBody is one practice page. text is the plain vim buffer.
func PageBody(name, text string) string {
	name = cleanName(name)
	if name == "" {
		name = "Session"
	}
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(name)
	b.WriteString("\n\n## ")
	b.WriteString(name)
	b.WriteString("\n\n")
	for _, line := range practiceLines(text) {
		b.WriteString("    ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// LessonText is the plain practice text of one session.
func LessonText(body string, partIndex, sectionIndex, lessonIndex int) (string, error) {
	lines, err := headingLines(body)
	if err != nil {
		return "", err
	}
	at, end, ok := lessonRegion(lines, partIndex, sectionIndex, lessonIndex)
	if !ok {
		return "", fmt.Errorf("session not found")
	}
	return practiceText(lines[at+1 : end]), nil
}

// SetLessonText replaces one session's practice text.
func SetLessonText(body string, partIndex, sectionIndex, lessonIndex int, text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("missing text")
	}
	lines, err := headingLines(body)
	if err != nil {
		return "", err
	}
	at, end, ok := lessonRegion(lines, partIndex, sectionIndex, lessonIndex)
	if !ok {
		return "", fmt.Errorf("session not found")
	}
	replacement := []string{""}
	for _, line := range practiceLines(text) {
		replacement = append(replacement, "    "+line)
	}
	out := append([]string{}, lines[:at+1]...)
	out = append(out, replacement...)
	out = append(out, lines[end:]...)
	return joinLines(out), nil
}

func lessonRegion(lines []string, partIndex, sectionIndex, lessonIndex int) (int, int, bool) {
	at, level, ok := findLesson(lines, partIndex, sectionIndex, lessonIndex)
	if !ok {
		return 0, 0, false
	}
	end := len(lines)
	for i := at + 1; i < len(lines); i++ {
		next, _, isHeading := headingTitle(lines[i])
		if isHeading && next <= level {
			end = i
			break
		}
	}
	return at, end, true
}

func practiceText(lines []string) string {
	var content []string
	started := false
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "    "):
			started = true
			content = append(content, strings.TrimRight(line[4:], " \t"))
		case strings.HasPrefix(line, "\t"):
			started = true
			content = append(content, strings.TrimRight(line[1:], " \t"))
		case started && strings.TrimSpace(line) == "":
			content = append(content, "")
		case started:
			started = false
		}
	}
	return strings.Join(trimBlankTail(content), "\n")
}

func trimBlankTail(lines []string) []string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func practiceLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return lines
}
