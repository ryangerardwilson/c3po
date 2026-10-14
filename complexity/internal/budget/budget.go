// Package budget says, before an edit, what each function can still afford.
package budget

import (
	"fmt"
	"strings"

	"github.com/ryangerardwilson/c3po/complexity/internal/accept"
	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
)

// Helper is the advice for a function that has no room left.
const Helper = "a new branch goes in a helper"

// Score is one metric of one function. Ceiling is the highest score a
// later ratchet passes: the limit, a legacy score above it, or an
// accepted score.
type Score struct {
	Metric  string `json:"metric"`
	Score   int    `json:"score"`
	Ceiling int    `json:"ceiling"`
	Left    int    `json:"left"`
}

// Entry is one function's budget. Full lists metrics at or over a limit.
// Tightest is the metric with the least room when nothing is full.
type Entry struct {
	File     string  `json:"file"`
	Line     int     `json:"line"`
	Name     string  `json:"name"`
	Full     []Score `json:"full,omitempty"`
	Tightest *Score  `json:"tightest,omitempty"`
	Action   string  `json:"action,omitempty"`
}

// Of computes budgets. stored and shown are the same functions; stored
// paths find accepted scores and shown paths are printed.
func Of(stored, shown []metric.Function, th metric.Thresholds, book accept.Book) []Entry {
	out := make([]Entry, 0, len(shown))
	for i, fn := range shown {
		e := Entry{File: fn.File, Line: fn.Line, Name: fn.Name}
		for _, name := range metric.Names {
			addScore(&e, stored[i], name, th, book)
		}
		if len(e.Full) > 0 {
			e.Tightest = nil
			e.Action = Helper
		}
		out = append(out, e)
	}
	return out
}

func addScore(e *Entry, fn metric.Function, name string, th metric.Thresholds, book accept.Book) {
	limit := metric.Limit(th, name)
	if limit <= 0 {
		return
	}
	s := Score{Metric: name, Score: metric.Value(fn, name), Ceiling: limit}
	if s.Score > s.Ceiling {
		s.Ceiling = s.Score
	}
	if rec, ok := book.Find(fn.File, fn.Name, name); ok && rec.Score > s.Ceiling {
		s.Ceiling = rec.Score
	}
	s.Left = s.Ceiling - s.Score
	if s.Score >= limit {
		e.Full = append(e.Full, s)
		return
	}
	if e.Tightest == nil || share(s, limit) < share(*e.Tightest, metric.Limit(th, e.Tightest.Metric)) {
		cp := s
		e.Tightest = &cp
	}
}

// share is the fraction of the limit still free, in thousandths.
func share(s Score, limit int) int {
	return s.Left * 1000 / limit
}

// Text renders one entry.
func (e Entry) Text() string {
	head := fmt.Sprintf("%s:%d %s", e.File, e.Line, e.Name)
	if len(e.Full) == 0 {
		if e.Tightest == nil {
			return head + ": no limits set"
		}
		t := *e.Tightest
		return fmt.Sprintf("%s: %s %d of %d, %d left", head, t.Metric, t.Score, t.Ceiling, t.Left)
	}
	parts := make([]string, 0, len(e.Full))
	for _, s := range e.Full {
		parts = append(parts, fmt.Sprintf("%s %d, ceiling %d", s.Metric, s.Score, s.Ceiling))
	}
	return fmt.Sprintf("%s: %s; %s", head, strings.Join(parts, "; "), e.Action)
}
