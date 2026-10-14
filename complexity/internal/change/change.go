// Package change states, function by function, what an edit made new and
// over a limit or worse than its baseline, and which score passes.
package change

import (
	"fmt"
	"sort"

	"github.com/ryangerardwilson/c3po/complexity/internal/accept"
	"github.com/ryangerardwilson/c3po/complexity/internal/dup"
	"github.com/ryangerardwilson/c3po/complexity/internal/gate"
	"github.com/ryangerardwilson/c3po/complexity/internal/linediff"
	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
	"github.com/ryangerardwilson/c3po/complexity/internal/models"
)

// Statuses of a Fact. Accepted is informational and does not fail.
const (
	New      = "new"
	Worse    = "worse"
	Accepted = "accepted"
)

// Fact is one metric of one function that is new and over its limit, or
// worse than its baseline. Line, Kind, Nesting, and Points name the added
// construct that accounts for the increase. Line is 0 when none is seen.
type Fact struct {
	File         string         `json:"file"`
	Name         string         `json:"name"`
	FunctionLine int            `json:"function_line"`
	Status       string         `json:"status"`
	Metric       string         `json:"metric"`
	Before       int            `json:"before"`
	After        int            `json:"after"`
	Delta        int            `json:"delta"`
	PassAt       int            `json:"pass_at"`
	Line         int            `json:"line"`
	Kind         string         `json:"kind"`
	Nesting      int            `json:"nesting"`
	Points       int            `json:"points"`
	Accepted     *accept.Record `json:"accepted,omitempty"`
	Action       string         `json:"action"`
	fresh        bool
}

// ModelFact is a model that is new and breaks a rule, or got worse.
type ModelFact struct {
	File       string   `json:"file"`
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	Violations []string `json:"violations"`
	Action     string   `json:"action"`
}

// Dup is a changed run of lines that copies existing code. It never fails.
type Dup struct {
	dup.Hit
	Action string `json:"action"`
}

// Report is everything an edit changed, compared with Ref.
type Report struct {
	Ref        string      `json:"ref"`
	Changes    []Fact      `json:"changes"`
	Models     []ModelFact `json:"models,omitempty"`
	Duplicates []Dup       `json:"duplicates,omitempty"`
}

// Input is one comparison. Maps holds, by file, how current lines map to
// baseline lines. A file without a map has no baseline text.
type Input struct {
	Cur, Base []metric.Function
	Maps      map[string]linediff.Map
	Th        metric.Thresholds
	Accepted  accept.Book
}

// Build compares cur with base and returns the facts plus the functions
// that fail, with Violations set to the failing metrics.
func Build(in Input) ([]Fact, []metric.Function) {
	var facts []Fact
	var bad []metric.Function
	for _, p := range gate.Match(in.Cur, in.Base, in.back) {
		got := pairFacts(p, in)
		facts = append(facts, got...)
		if names := failing(got); len(names) > 0 {
			fn := p.Cur
			fn.Violations = names
			bad = append(bad, fn)
		}
	}
	sort.SliceStable(facts, func(i, j int) bool { return before(facts[i], facts[j]) })
	return facts, bad
}

func (in Input) back(file string, line int) (int, bool) {
	m, ok := in.Maps[file]
	if !ok {
		return line, true
	}
	return m.Old(line)
}

func before(a, b Fact) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	return a.FunctionLine < b.FunctionLine
}

func failing(facts []Fact) []string {
	var out []string
	for _, f := range facts {
		if f.Status != Accepted {
			out = append(out, f.Metric)
		}
	}
	return out
}

func pairFacts(p gate.Pair, in Input) []Fact {
	var out []Fact
	for _, name := range metric.Names {
		if f, ok := oneFact(p, name, in); ok {
			out = append(out, f)
		}
	}
	return out
}

func oneFact(p gate.Pair, name string, in Input) (Fact, bool) {
	limit := metric.Limit(in.Th, name)
	after := metric.Value(p.Cur, name)
	pass := gate.PassAt(limit, p.Base, name)
	if limit <= 0 || after <= pass {
		return Fact{}, false
	}
	f := Fact{
		File: p.Cur.File, Name: p.Cur.Name, FunctionLine: p.Cur.Line,
		Status: New, Metric: name, After: after, PassAt: pass,
	}
	f.fresh = p.Base == nil
	if !f.fresh {
		f.Status = Worse
		f.Before = metric.Value(*p.Base, name)
	}
	f.Delta = f.After - f.Before
	applyAccepted(&f, in.Accepted)
	attribute(&f, p, in.fresh(p))
	f.Action = action(f)
	return f, true
}

// applyAccepted raises the passing score to an accepted one. A score at or
// under it is reported as accepted instead of failing.
func applyAccepted(f *Fact, book accept.Book) {
	rec, ok := book.Find(f.File, f.Name, f.Metric)
	if !ok || rec.Score < f.PassAt {
		return
	}
	f.Accepted = &rec
	f.PassAt = rec.Score
	if f.After <= rec.Score {
		f.Status = Accepted
	}
}

// fresh reports which lines of the current function are new. All lines of
// a new function are new. Without baseline text, nothing is attributed.
func (in Input) fresh(p gate.Pair) func(int) bool {
	if p.Base == nil {
		return func(int) bool { return true }
	}
	m, ok := in.Maps[p.Cur.File]
	if !ok {
		return nil
	}
	return m.Fresh
}

// attribute names the hottest construct the edit added: a scored event on
// a new line that the baseline function did not already have. When every
// new-line event matches an old one, the hottest new-line event is named.
func attribute(f *Fact, p gate.Pair, fresh func(int) bool) {
	if fresh == nil {
		return
	}
	if !pick(f, addedEvents(p, fresh)) {
		pick(f, freshEvents(p.Cur.Events, fresh))
	}
}

type eventKey struct {
	kind                       string
	nesting, cog, cyclo, depth int
}

func keyOf(e metric.Event) eventKey {
	return eventKey{e.Kind, e.Nesting, e.Cog, e.Cyclo, e.Depth}
}

// addedEvents removes events the baseline already had, consuming old-line
// events first, and returns what is left on new lines.
func addedEvents(p gate.Pair, fresh func(int) bool) []metric.Event {
	left := map[eventKey]int{}
	if p.Base != nil {
		for _, e := range p.Base.Events {
			left[keyOf(e)]++
		}
	}
	var out []metric.Event
	for _, onNew := range []bool{false, true} {
		for _, e := range p.Cur.Events {
			if fresh(e.Line) != onNew {
				continue
			}
			if k := keyOf(e); left[k] > 0 {
				left[k]--
			} else if onNew {
				out = append(out, e)
			}
		}
	}
	return out
}

func freshEvents(events []metric.Event, fresh func(int) bool) []metric.Event {
	var out []metric.Event
	for _, e := range events {
		if fresh(e.Line) {
			out = append(out, e)
		}
	}
	return out
}

// pick sets f's construct to the first event with the most points.
func pick(f *Fact, events []metric.Event) bool {
	found := false
	for _, e := range events {
		pts := e.Points(f.Metric)
		if pts > 0 && pts > f.Points {
			f.Line, f.Kind, f.Nesting, f.Points = e.Line, e.Kind, e.Nesting, pts
			found = true
		}
	}
	return found
}

// Failing reports whether a report holds a new or worse function or model.
func (r Report) Failing() bool {
	return len(failing(r.Changes)) > 0 || len(r.Models) > 0
}

// DupFacts turns duplicate hits into informational facts.
func DupFacts(hits []dup.Hit) []Dup {
	out := make([]Dup, 0, len(hits))
	for _, h := range hits {
		out = append(out, Dup{Hit: h, Action: dupAction(h)})
	}
	return out
}

// ModelFacts states each regressed model as new or worse.
func ModelFacts(bad, base []models.Model) []ModelFact {
	known := map[string]bool{}
	for _, m := range base {
		known[m.File+"\x00"+m.Name] = true
	}
	out := make([]ModelFact, 0, len(bad))
	for _, m := range bad {
		f := ModelFact{File: m.File, Name: m.Name, Status: New, Violations: m.Violations}
		f.Action = fmt.Sprintf("Bring %s within the model rules before you finish.", m.Name)
		if known[m.File+"\x00"+m.Name] {
			f.Status = Worse
			f.Action = "Undo the change that made it worse."
		}
		out = append(out, f)
	}
	return out
}
