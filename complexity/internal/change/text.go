package change

import (
	"fmt"
	"strings"

	"github.com/ryangerardwilson/c3po/complexity/internal/dup"
)

func action(f Fact) string {
	switch f.Status {
	case New:
		return fmt.Sprintf("Get it under %d.", f.PassAt)
	case Accepted:
		return fmt.Sprintf("Accepted at %d on record: %s", f.Accepted.Score, f.Accepted.Reason)
	default:
		return "Undo or offset that increase."
	}
}

func dupAction(h dup.Hit) string {
	return fmt.Sprintf("duplicates %s:%d; call it instead of copying", h.OfFile, h.OfLine)
}

// Counts returns how many functions are new, worse, and accepted.
func (r Report) Counts() (fresh, worse, accepted int) {
	for _, g := range groups(r.Changes) {
		switch g[0].Status {
		case New:
			fresh++
		case Worse:
			worse++
		default:
			accepted++
		}
	}
	return fresh, worse, accepted
}

// Lines renders the report: a file path, then one indented line per
// function, model, or duplicate in that file.
func (r Report) Lines() []string {
	byFile := map[string][]string{}
	var order []string
	add := func(file, line string) {
		if _, ok := byFile[file]; !ok {
			order = append(order, file)
		}
		byFile[file] = append(byFile[file], "  "+line)
	}
	for _, g := range groups(r.Changes) {
		add(g[0].File, Sentence(g))
	}
	for _, m := range r.Models {
		add(m.File, modelSentence(m))
	}
	for _, d := range r.Duplicates {
		add(d.File, fmt.Sprintf("L%d %s", d.Line, d.Action))
	}
	var out []string
	for _, file := range order {
		out = append(out, file)
		out = append(out, byFile[file]...)
	}
	return out
}

// groups splits facts into one run per function and status class.
func groups(facts []Fact) [][]Fact {
	var out [][]Fact
	for _, f := range facts {
		n := len(out)
		if n > 0 && sameGroup(out[n-1][0], f) {
			out[n-1] = append(out[n-1], f)
			continue
		}
		out = append(out, []Fact{f})
	}
	return out
}

func sameGroup(a, b Fact) bool {
	return a.File == b.File && a.Name == b.Name && a.FunctionLine == b.FunctionLine &&
		(a.Status == Accepted) == (b.Status == Accepted)
}

// Sentence renders one function's facts on one line.
func Sentence(g []Fact) string {
	switch g[0].Status {
	case New:
		return newSentence(g)
	case Accepted:
		return acceptedSentence(g)
	default:
		return worseSentence(g)
	}
}

func newSentence(g []Fact) string {
	f := g[0]
	s := fmt.Sprintf("%s (new) %s %d > %d", f.Name, f.Metric, f.After, f.PassAt)
	if f.Kind != "" {
		s += fmt.Sprintf(": %s at L%d, nesting %d, +%d", f.Kind, f.Line, f.Nesting, f.Points)
	}
	var more []string
	for _, o := range g[1:] {
		s += fmt.Sprintf("; %s %d > %d", o.Metric, o.After, o.PassAt)
		more = append(more, fmt.Sprintf("%s under %d", o.Metric, o.PassAt))
	}
	if len(more) == 0 {
		return s + ". " + f.Action
	}
	return fmt.Sprintf("%s. Get it under %d, %s.", s, f.PassAt, strings.Join(more, ", "))
}

func worseSentence(g []Fact) string {
	parts := make([]string, 0, len(g))
	for _, f := range g {
		parts = append(parts, fmt.Sprintf("%s %d→%d (%s); passes at ≤%d%s", f.Metric, f.Before, f.After, cause(f), f.PassAt, onRecord(f)))
	}
	return g[0].Name + " " + strings.Join(parts, "; ") + ". " + g[0].Action
}

func acceptedSentence(g []Fact) string {
	parts := make([]string, 0, len(g))
	for _, f := range g {
		parts = append(parts, fmt.Sprintf("%s %s", f.Metric, change(f)))
	}
	return fmt.Sprintf("%s %s; accepted on record at ≤%d: %s", g[0].Name, strings.Join(parts, "; "), g[0].PassAt, g[0].Accepted.Reason)
}

func change(f Fact) string {
	if f.fresh {
		return fmt.Sprintf("%d (new)", f.After)
	}
	return fmt.Sprintf("%d→%d (%s)", f.Before, f.After, cause(f))
}

func cause(f Fact) string {
	s := fmt.Sprintf("%+d", f.Delta)
	if f.Kind != "" {
		s += fmt.Sprintf(", %s at L%d, nesting %d", f.Kind, f.Line, f.Nesting)
	}
	return s
}

func onRecord(f Fact) string {
	if f.Accepted == nil {
		return ""
	}
	return " (accepted score on record)"
}

func modelSentence(m ModelFact) string {
	return fmt.Sprintf("%s model (%s) %s. %s", m.Name, m.Status, strings.Join(m.Violations, ", "), m.Action)
}
