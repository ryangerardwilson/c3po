package metric

import "sort"

// Function is one scored unit of code.
type Function struct {
	File       string   `json:"file"`
	Line       int      `json:"line"`
	EndLine    int      `json:"end_line"`
	Name       string   `json:"name"`
	Language   string   `json:"language"`
	Cognitive  int      `json:"cognitive"`
	Nesting    int      `json:"nesting"`
	Lines      int      `json:"lines"`
	Params     int      `json:"params"`
	Cyclomatic int      `json:"cyclomatic"`
	Violations []string `json:"violations,omitempty"`
	// Sig identifies an anonymous function by its body inside one file.
	Sig string `json:"-"`
	// Events are the scored constructs that attribute an increase to a line.
	Events []Event `json:"-"`
}

// Thresholds are the maximums a function may reach.
// Zero disables that metric.
type Thresholds struct {
	Cognitive  int `json:"cognitive" toml:"cognitive"`
	Nesting    int `json:"nesting" toml:"nesting"`
	Lines      int `json:"lines" toml:"lines"`
	Params     int `json:"params" toml:"params"`
	Cyclomatic int `json:"cyclomatic" toml:"cyclomatic"`
}

// Violations lists metrics that are above a positive threshold.
func Violations(fn Function, th Thresholds) []string {
	var out []string
	out = appendOver(out, "cognitive", th.Cognitive, fn.Cognitive)
	out = appendOver(out, "nesting", th.Nesting, fn.Nesting)
	out = appendOver(out, "lines", th.Lines, fn.Lines)
	out = appendOver(out, "params", th.Params, fn.Params)
	out = appendOver(out, "cyclomatic", th.Cyclomatic, fn.Cyclomatic)
	return out
}

func appendOver(out []string, name string, limit, value int) []string {
	if limit > 0 && value > limit {
		return append(out, name)
	}
	return out
}

// Mark sets Violations from th.
func (fn *Function) Mark(th Thresholds) {
	fn.Violations = Violations(*fn, th)
}

// Percentile returns the nearest-rank percentile of a sorted or unsorted
// slice. p is in the range 0 to 100. An empty slice returns 0.
func Percentile(values []int, p int) int {
	if len(values) == 0 {
		return 0
	}
	cp := append([]int(nil), values...)
	sort.Ints(cp)
	if p <= 0 {
		return cp[0]
	}
	if p >= 100 {
		return cp[len(cp)-1]
	}
	rank := (p*len(cp) + 99) / 100
	if rank < 1 {
		rank = 1
	}
	if rank > len(cp) {
		rank = len(cp)
	}
	return cp[rank-1]
}
