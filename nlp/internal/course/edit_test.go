package course

import (
	"strings"
	"testing"
)

const sample = `# Demo

## Part I: Alpha

### Section 1: One

#### Lesson 1: A

    a

### Section 2: Two

#### Lesson 1: B

    b

## Part II: Beta

### Section A: Aye

#### Lesson 1: C

    c

### Section B: Bee

#### Lesson 1: D

    d

## Part III: Gamma

### Section 1: Gee

#### Lesson 1: E

    e
`

func TestDeletePartClosesRomanNumbers(t *testing.T) {
	body, err := DeletePart(sample, 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "Part I: Alpha") {
		t.Fatalf("deleted part remained:\n%s", body)
	}
	if !strings.Contains(body, "## Part I: Beta") {
		t.Fatalf("Part II was not renumbered to Part I:\n%s", body)
	}
	if !strings.Contains(body, "## Part II: Gamma") {
		t.Fatalf("Part III was not renumbered to Part II:\n%s", body)
	}
	if !strings.Contains(body, "### Section A: Aye") {
		t.Fatalf("section letter was rewritten:\n%s", body)
	}
	parsed, ok := ParseMarkdown("demo", body)
	if !ok || len(parsed.Parts) != 2 {
		t.Fatalf("parsed parts = %d ok=%v", len(parsed.Parts), ok)
	}
}

func TestDeleteSectionClosesNumbersAndLetters(t *testing.T) {
	body, err := DeleteSection(sample, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "Section 1: One") || !strings.Contains(body, "### Section 1: Two") {
		t.Fatalf("numeric sections did not close:\n%s", body)
	}
	letters, err := DeleteSection(sample, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(letters, "Section A: Aye") || !strings.Contains(letters, "### Section A: Bee") {
		t.Fatalf("letter sections did not close:\n%s", letters)
	}
	if !strings.Contains(letters, "## Part II: Beta") {
		t.Fatalf("part numbers changed during a section delete:\n%s", letters)
	}
}

func TestInsertPartAndSectionUseNextNumber(t *testing.T) {
	body, err := InsertPart(sample, "Delta", 2, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "## Part IV: Delta") {
		t.Fatalf("new part number:\n%s", body)
	}
	body, err = InsertSection(sample, 0, 1, "Three")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "### Section 3: Three") {
		t.Fatalf("new section number:\n%s", body)
	}
	body, err = InsertSection(sample, 1, 1, "Cee")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "### Section C: Cee") {
		t.Fatalf("new section letter:\n%s", body)
	}
}

func TestRenameKeepsNumberTextAsTyped(t *testing.T) {
	body, err := RenamePart(sample, 1, "Part II: Tools")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "## Part II: Tools") || strings.Contains(body, "Part II: Beta") {
		t.Fatalf("rename:\n%s", body)
	}
	body, err = RenameCourse(body, "Practice")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "# Practice\n") {
		t.Fatalf("course rename:\n%s", body)
	}
}

func TestNewCourseParses(t *testing.T) {
	body := NewCourseBody("Widgets")
	parsed, ok := ParseMarkdown("widgets", body)
	if !ok {
		t.Fatal("new course did not parse")
	}
	if parsed.Name != "Widgets" || len(parsed.Parts) != 1 || len(parsed.Parts[0].Sections) != 1 {
		t.Fatalf("parsed %#v", parsed)
	}
}

func TestRefuseLastPart(t *testing.T) {
	body := NewCourseBody("Solo")
	if _, err := DeletePart(body, 0); err == nil {
		t.Fatal("deleted the only part")
	}
}

func TestInsertStripsTypedNumberPrefix(t *testing.T) {
	body, err := InsertPart(sample, "Part IX: Delta", 2, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "## Part IV: Delta") || strings.Contains(body, "Part IX") {
		t.Fatalf("part prefix was kept:\n%s", body)
	}
	body, err = InsertSection(sample, 0, 1, "Section 9: Three")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "### Section 3: Three") || strings.Contains(body, "Section 9") {
		t.Fatalf("section prefix was kept:\n%s", body)
	}
}

func TestInsertAfterCursorRenumbersFollowing(t *testing.T) {
	body, err := InsertPart(sample, "Delta", 0, true)
	if err != nil {
		t.Fatal(err)
	}
	parsed, ok := ParseMarkdown("demo", body)
	if !ok || len(parsed.Parts) != 4 {
		t.Fatalf("parts = %d ok=%v", len(parsed.Parts), ok)
	}
	got := []string{parsed.Parts[0].Name, parsed.Parts[1].Name, parsed.Parts[2].Name, parsed.Parts[3].Name}
	want := []string{"Part I: Alpha", "Part II: Delta", "Part III: Beta", "Part IV: Gamma"}
	if strings.Join(got, " | ") != strings.Join(want, " | ") {
		t.Fatalf("parts = %v", got)
	}
	if parsed.Parts[2].Sections[0].Name != "Section A: Aye" {
		t.Fatalf("later section changed: %q", parsed.Parts[2].Sections[0].Name)
	}

	body, err = InsertSection(sample, 0, 0, "Three")
	if err != nil {
		t.Fatal(err)
	}
	parsed, ok = ParseMarkdown("demo", body)
	if !ok {
		t.Fatal("section insert did not parse")
	}
	got = []string{parsed.Parts[0].Sections[0].Name, parsed.Parts[0].Sections[1].Name, parsed.Parts[0].Sections[2].Name}
	want = []string{"Section 1: One", "Section 2: Three", "Section 3: Two"}
	if strings.Join(got, " | ") != strings.Join(want, " | ") {
		t.Fatalf("sections = %v", got)
	}

	body, err = InsertSection(sample, 1, 0, "Cee")
	if err != nil {
		t.Fatal(err)
	}
	parsed, ok = ParseMarkdown("demo", body)
	if !ok {
		t.Fatal("letter insert did not parse")
	}
	got = []string{parsed.Parts[1].Sections[0].Name, parsed.Parts[1].Sections[1].Name, parsed.Parts[1].Sections[2].Name}
	want = []string{"Section A: Aye", "Section B: Cee", "Section C: Bee"}
	if strings.Join(got, " | ") != strings.Join(want, " | ") {
		t.Fatalf("letters = %v", got)
	}
	if parsed.Parts[1].Name != "Part II: Beta" {
		t.Fatalf("part number changed: %q", parsed.Parts[1].Name)
	}
}

func TestRenameAndDeleteLessonPage(t *testing.T) {
	body, err := RenameLesson(sample, 0, 0, 0, "Prayer")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "#### Prayer") || strings.Contains(body, "#### Lesson 1: A") {
		t.Fatalf("renamed lesson:\n%s", body)
	}
	body, err = DeleteLesson(body, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "Prayer") {
		t.Fatalf("deleted lesson remained:\n%s", body)
	}
	if _, err := DeleteLesson(NewCourseBody("Solo"), 0, 0, 0); err == nil {
		t.Fatal("deleted the only session")
	}
}
func TestPageBodyAndLessonText(t *testing.T) {
	body := PageBody("Prayer", "alpha\n\nbeta")
	parsed, ok := ParseMarkdown("prayer", body)
	if !ok || parsed.Name != "Prayer" || len(parsed.Parts) != 1 {
		t.Fatalf("parsed %#v ok=%v", parsed, ok)
	}
	got, err := LessonText(body, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != "alpha\n\nbeta" {
		t.Fatalf("text = %q", got)
	}
	body, err = SetLessonText(sample, 0, 0, 0, "aye\nstill")
	if err != nil {
		t.Fatal(err)
	}
	got, err = LessonText(body, 0, 0, 0)
	if err != nil || got != "aye\nstill" {
		t.Fatalf("edited = %q err=%v", got, err)
	}
	other, err := LessonText(body, 0, 1, 0)
	if err != nil || other != "b" {
		t.Fatalf("other lesson = %q err=%v\n%s", other, err, body)
	}
	if _, err := SetLessonText(body, 0, 0, 0, " \n\t"); err == nil {
		t.Fatal("stored blank practice text")
	}
	flat, err := LessonText(NewCourseBody("Solo"), 0, 0, 0)
	if err != nil || flat != "..." {
		t.Fatalf("flat = %q err=%v", flat, err)
	}
}
