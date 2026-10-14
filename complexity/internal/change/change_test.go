package change

import (
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/complexity/internal/accept"
	"github.com/ryangerardwilson/c3po/complexity/internal/linediff"
	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
	"github.com/ryangerardwilson/c3po/complexity/internal/parse"
)

var th = metric.Thresholds{Cognitive: 15, Nesting: 4, Lines: 80, Params: 6, Cyclomatic: 0}

func build(t *testing.T, oldSrc, newSrc string, book accept.Book) Report {
	t.Helper()
	base, _ := parse.File("a.py", "python", []byte(oldSrc))
	cur, _ := parse.File("a.py", "python", []byte(newSrc))
	maps := map[string]linediff.Map{"a.py": linediff.Lines(oldSrc, newSrc)}
	facts, _ := Build(Input{Cur: cur, Base: base, Maps: maps, Th: th, Accepted: book})
	return Report{Ref: "HEAD", Changes: facts}
}

// legacy has cognitive 25 and reaches nesting 5 at its deepest if.
// Its last line is 27.
const legacy = `def legacy(a):
    if a:
        if a:
            if a:
                if a:
                    if a:
                        pass
                    elif a:
                        pass
    if a:
        pass
    if a:
        pass
    if a:
        pass
    if a:
        pass
    if a:
        pass
    if a:
        pass
    if a:
        pass
    if a:
        pass
    if a:
        pass
`

func TestWorseLegacyPassesAtPreviousScore(t *testing.T) {
	cur := strings.Replace(legacy, "                    elif a:\n                        pass\n",
		"                    elif a:\n                        pass\n                    elif a:\n                        pass\n", 1)
	r := build(t, legacy, cur, accept.Book{})
	if len(r.Changes) != 1 {
		t.Fatalf("%+v", r.Changes)
	}
	f := r.Changes[0]
	if f.Status != Worse || f.Metric != "cognitive" || f.Before != 25 || f.After != 26 || f.PassAt != 25 {
		t.Fatalf("%+v", f)
	}
	if f.Kind != "elif" || f.Line != 10 || f.Nesting != 4 {
		t.Fatalf("construct %+v", f)
	}
	want := "legacy cognitive 25→26 (+1, elif at L10, nesting 4); passes at ≤25. Undo or offset that increase."
	if got := Sentence(r.Changes); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestNewFunctionNamesHottestConstruct(t *testing.T) {
	cur := legacy + "\ndef handle(a):\n" + strings.Repeat("    if a:\n        pass\n", 4) +
		"    if a:\n        if a:\n            if a:\n                if a:\n                    if a:\n                        pass\n"
	r := build(t, legacy, cur, accept.Book{})
	var got []string
	for _, g := range groups(r.Changes) {
		got = append(got, Sentence(g))
	}
	want := "handle (new) cognitive 19 > 15: if at L42, nesting 4, +5; nesting 5 > 4. Get it under 15, nesting under 4."
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestUnchangedLegacyIsSilent(t *testing.T) {
	r := build(t, legacy, "# moved\n"+legacy, accept.Book{})
	if len(r.Changes) != 0 {
		t.Fatalf("%+v", r.Changes)
	}
}

func TestAcceptedScoreIsTheCeiling(t *testing.T) {
	cur := strings.Replace(legacy, "    if a:\n        pass\n", "    if a and a:\n        pass\n", 1)
	book := accept.Book{Accepted: []accept.Record{{File: "a.py", Function: "legacy", Metric: "cognitive", Score: 26, Reason: "parser table"}}}
	r := build(t, legacy, cur, book)
	if r.Failing() || len(r.Changes) != 1 || r.Changes[0].Status != Accepted || r.Changes[0].Kind != "and" {
		t.Fatalf("%+v", r.Changes)
	}
	if line := strings.Join(r.Lines(), "\n"); !strings.Contains(line, "accepted on record at ≤26: parser table") {
		t.Fatalf("%s", line)
	}
	worse := strings.Replace(cur, "    if a:\n        pass\n", "    if a and a:\n        pass\n", 1)
	r = build(t, legacy, worse, book)
	if !r.Failing() || r.Changes[0].PassAt != 26 {
		t.Fatalf("%+v", r.Changes)
	}
}
