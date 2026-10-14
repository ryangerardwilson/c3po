package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
	"github.com/ryangerardwilson/c3po/complexity/internal/models"
)

func writeText(w io.Writer, doc Document, fns []metric.Function, view View) error {
	writeChanges(w, doc, view)
	if writeList(w, doc, fns, view) && !view.Quiet {
		fmt.Fprintln(w)
	}
	if !view.Quiet {
		writeSummary(w, doc)
		writeModels(w, doc)
	}
	if view.ShowExit {
		fmt.Fprintf(w, "\nExit code: %d\n", view.ExitCode)
	}
	return nil
}

func writeList(w io.Writer, doc Document, fns []metric.Function, view View) bool {
	list, title, ok := listSection(fns, doc, view)
	if !ok {
		return false
	}
	shown, extra := clipList(list, view.Top)
	if !view.Quiet {
		fmt.Fprintln(w, title)
		fmt.Fprintln(w, strings.Repeat("─", 72))
	}
	for _, fn := range shown {
		writeFn(w, fn)
	}
	if extra > 0 && !view.Quiet {
		fmt.Fprintf(w, "… and %d more\n", extra)
	}
	return true
}

// writeChanges prints what got new or worse, one function per line, before
// the hotspot list.
func writeChanges(w io.Writer, doc Document, view View) {
	if doc.Ratchet == nil || view.RankBy != "" {
		return
	}
	lines := doc.Ratchet.Lines()
	if len(lines) == 0 {
		return
	}
	if !view.Quiet {
		fmt.Fprintf(w, "Changes vs %s\n%s\n", doc.Ratchet.Ref, strings.Repeat("─", 72))
	}
	fmt.Fprintln(w, strings.Join(lines, "\n"))
	if !view.Quiet {
		fmt.Fprintln(w)
	}
}

func listSection(fns []metric.Function, doc Document, view View) ([]metric.Function, string, bool) {
	if view.RankBy != "" {
		list := doc.Ranked
		if list == nil {
			list = Order(fns, view.RankBy)
		}
		return list, "Rank by " + view.RankBy, true
	}
	if len(doc.Hotspots) == 0 {
		return nil, "", false
	}
	return doc.Hotspots, "Hotspots", true
}

func writeFn(w io.Writer, fn metric.Function) {
	fmt.Fprintf(w, "%s:%d\t%s\tcog=%d  nest=%d  lines=%d  params=%d  cyclo=%d\n",
		fn.File, fn.Line, fn.Name, fn.Cognitive, fn.Nesting, fn.Lines, fn.Params, fn.Cyclomatic)
}

func clipList(list []metric.Function, n int) ([]metric.Function, int) {
	if n <= 0 {
		n = 20
	}
	if len(list) <= n {
		return list, 0
	}
	return list[:n], len(list) - n
}

func writeSummary(w io.Writer, doc Document) {
	fmt.Fprintln(w, "Summary")
	fmt.Fprintln(w, strings.Repeat("─", 72))
	fmt.Fprintf(w, "Functions analyzed          %d\n", doc.Summary.FunctionsAnalyzed)
	fmt.Fprintf(w, "Hotspots                    %d\n", doc.Summary.Hotspots)
	fmt.Fprintf(w, "p90 / p95 cognitive         %d / %d\n", doc.Summary.P90Cognitive, doc.Summary.P95Cognitive)
	fmt.Fprintf(w, "Max cognitive               %d\n", doc.Summary.MaxCognitive)
	fmt.Fprintf(w, "Max nesting                 %d\n", doc.Summary.MaxNesting)
	fmt.Fprintf(w, "Functions > %d lines        %d\n", lineLimit(doc), doc.Summary.LongFunctions)
	fmt.Fprintf(w, "Duplication clusters        %d   (~%d lines)\n", doc.Summary.DuplicationClusters, doc.Summary.DuplicatedLinesApprox)
}

func lineLimit(doc Document) int {
	if doc.Thresholds.Lines > 0 {
		return doc.Thresholds.Lines
	}
	return 80
}

func writeModels(w io.Writer, doc Document) {
	if doc.Models == nil || (len(doc.Models.Hotspots) == 0 && doc.Models.Summary.ModelsAnalyzed == 0) {
		return
	}
	writeModelHotspots(w, doc.Models.Hotspots)
	s := doc.Models.Summary
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Models")
	fmt.Fprintln(w, strings.Repeat("─", 72))
	fmt.Fprintf(w, "Models analyzed             %d\n", s.ModelsAnalyzed)
	fmt.Fprintf(w, "High-degree models          %d\n", s.HighDegree)
	fmt.Fprintf(w, "Deepest referential path    %d\n", s.MaxDepth)
	fmt.Fprintf(w, "Cycles detected             %d\n", s.Cycles)
	fmt.Fprintf(w, "Total M2M relationships     %d\n", s.TotalM2M)
	writeConventions(w, doc)
}

func writeModelHotspots(w io.Writer, hot []models.Model) {
	if len(hot) == 0 {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Model Relationship Hotspots")
	fmt.Fprintln(w, strings.Repeat("─", 72))
	for _, m := range hot {
		cycles := 0
		if m.Cycles {
			cycles = 1
		}
		fmt.Fprintf(w, "%-20s degree=%-3d depth=%-2d m2m=%-2d cycles=%d   %s\n",
			m.Name, m.Degree, m.Depth, m.M2M, cycles, m.File)
	}
}

func writeConventions(w io.Writer, doc Document) {
	if !hasConvention(doc) {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Convention Violations")
	fmt.Fprintln(w, strings.Repeat("─", 72))
	for _, m := range doc.Models.Hotspots {
		writeOneConvention(w, m)
	}
}

func hasConvention(doc Document) bool {
	for _, m := range doc.Models.Hotspots {
		if conventionViolation(m.Violations) {
			return true
		}
	}
	return false
}

func conventionViolation(vs []string) bool {
	for _, v := range vs {
		if v == "naming" || v == "column_order" {
			return true
		}
	}
	return false
}

func writeOneConvention(w io.Writer, m models.Model) {
	for _, v := range m.Violations {
		if v != "naming" && v != "column_order" {
			continue
		}
		fmt.Fprintf(w, "%s:%d  %s\n  %s\n", m.File, m.Line, m.Name, v)
		writeSuggestions(w, m.Suggestions)
	}
}

func writeSuggestions(w io.Writer, suggestions []string) {
	for _, s := range suggestions {
		fmt.Fprintf(w, "  suggestion: %s\n", s)
	}
}

func ranked(fns []metric.Function) []metric.Function {
	return Order(fns, "cognitive")
}

func Order(fns []metric.Function, by string) []metric.Function {
	out := append([]metric.Function(nil), fns...)
	sort.Slice(out, func(i, j int) bool {
		return rankLess(out[i], out[j], by)
	})
	return out
}

func rankLess(a, b metric.Function, by string) bool {
	av, bv := metricValue(a, by), metricValue(b, by)
	if av != bv {
		return av > bv
	}
	if a.Lines != b.Lines {
		return a.Lines > b.Lines
	}
	if a.Name != b.Name {
		return a.Name < b.Name
	}
	if a.File != b.File {
		return a.File < b.File
	}
	return a.Line < b.Line
}

func metricValue(fn metric.Function, by string) int {
	switch by {
	case "nesting":
		return fn.Nesting
	case "lines":
		return fn.Lines
	case "params":
		return fn.Params
	case "cyclomatic":
		return fn.Cyclomatic
	default:
		return fn.Cognitive
	}
}
