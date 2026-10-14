package budget

import (
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/complexity/internal/accept"
	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
)

func TestBudgetLines(t *testing.T) {
	th := metric.Thresholds{Cognitive: 15, Nesting: 4, Lines: 80, Params: 6, Cyclomatic: 10}
	fns := []metric.Function{
		{File: "a.go", Line: 3, Name: "legacy", Cognitive: 25, Nesting: 2, Lines: 40, Params: 1, Cyclomatic: 9},
		{File: "a.go", Line: 40, Name: "edge", Cognitive: 15, Nesting: 1, Lines: 10, Params: 1, Cyclomatic: 4},
		{File: "a.go", Line: 60, Name: "small", Cognitive: 3, Nesting: 1, Lines: 70, Params: 1, Cyclomatic: 2},
	}
	book := accept.Book{Accepted: []accept.Record{{File: "a.go", Function: "legacy", Metric: "cognitive", Score: 27}}}
	got := Of(fns, fns, th, book)
	want := []string{
		"a.go:3 legacy: cognitive 25, ceiling 27; a new branch goes in a helper",
		"a.go:40 edge: cognitive 15, ceiling 15; a new branch goes in a helper",
		"a.go:60 small: lines 70 of 80, 10 left",
	}
	for i, e := range got {
		if e.Text() != want[i] {
			t.Fatalf("line %d\ngot  %s\nwant %s", i, e.Text(), want[i])
		}
	}
	if !strings.Contains(got[0].Text(), Helper) || got[2].Action != "" {
		t.Fatalf("%+v", got)
	}
}
