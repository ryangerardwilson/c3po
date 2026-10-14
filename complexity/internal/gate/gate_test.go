package gate

import (
	"testing"

	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
)

func TestLegacyHotspotStays(t *testing.T) {
	th := metric.Thresholds{Cognitive: 15, Nesting: 4, Lines: 80, Params: 6}
	old := metric.Function{File: "a.go", Name: "hot", Line: 10, Cognitive: 28, Nesting: 5, Lines: 90, Params: 7}
	cur := old
	if got := Regressions([]metric.Function{cur}, []metric.Function{old}, th); len(got) != 0 {
		t.Fatalf("%#v", got)
	}
	cur.Cognitive = 30
	got := Regressions([]metric.Function{cur}, []metric.Function{old}, th)
	if len(got) != 1 || got[0].Violations[0] != "cognitive" {
		t.Fatalf("%#v", got)
	}
}

func TestNewFunctionFails(t *testing.T) {
	th := metric.Thresholds{Cognitive: 15, Nesting: 4, Lines: 80, Params: 6}
	cur := metric.Function{File: "a.go", Name: "fresh", Line: 1, Cognitive: 16}
	got := Regressions([]metric.Function{cur}, nil, th)
	if len(got) != 1 {
		t.Fatalf("%#v", got)
	}
}
