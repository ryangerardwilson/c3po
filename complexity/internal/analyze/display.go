package analyze

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/complexity/internal/change"
	"github.com/ryangerardwilson/c3po/complexity/internal/config"
	"github.com/ryangerardwilson/c3po/complexity/internal/dup"
	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
	"github.com/ryangerardwilson/c3po/complexity/internal/models"
)

// ForDisplay returns a copy whose listed paths are relative to the working directory.
func (r Result) ForDisplay() Result {
	cwd, err := os.Getwd()
	if err != nil {
		return r
	}
	return r.relativeTo(cwd)
}

// ForDisplayAt returns a copy whose listed paths are relative to dir.
// An empty dir uses the process working directory.
func (r Result) ForDisplayAt(dir string) Result {
	if dir == "" {
		return r.ForDisplay()
	}
	return r.relativeTo(dir)
}

func (r Result) relativeTo(cwd string) Result {
	r.Functions = relFns(cwd, r.abs, r.Functions)
	r.Hotspots = relFns(cwd, r.abs, r.Hotspots)
	r.Clusters = relClusters(cwd, r.abs, r.Clusters)
	r.Models = relSection(cwd, r.abs, r.Models)
	r.Diagnostics = relDiags(cwd, r.abs, r.Diagnostics)
	r.Report = relReport(cwd, r.abs, r.Report)
	r.Files = relPaths(cwd, r.abs, r.Files)
	return r
}

func relPaths(cwd string, abs map[string]string, paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = relFile(cwd, abs, p)
	}
	return out
}

func relReport(cwd string, abs map[string]string, rep change.Report) change.Report {
	rep.Changes = append([]change.Fact(nil), rep.Changes...)
	for i := range rep.Changes {
		rep.Changes[i].File = relFile(cwd, abs, rep.Changes[i].File)
	}
	rep.Models = append([]change.ModelFact(nil), rep.Models...)
	for i := range rep.Models {
		rep.Models[i].File = relFile(cwd, abs, rep.Models[i].File)
	}
	hits := make([]dup.Hit, 0, len(rep.Duplicates))
	for _, d := range rep.Duplicates {
		h := d.Hit
		h.File = relFile(cwd, abs, h.File)
		h.OfFile = relFile(cwd, abs, h.OfFile)
		hits = append(hits, h)
	}
	if len(hits) > 0 {
		rep.Duplicates = change.DupFacts(hits)
	}
	return rep
}

func relFns(cwd string, abs map[string]string, fns []metric.Function) []metric.Function {
	out := append([]metric.Function(nil), fns...)
	for i := range out {
		out[i].File = relFile(cwd, abs, out[i].File)
	}
	return out
}

func relClusters(cwd string, abs map[string]string, clusters []dup.Cluster) []dup.Cluster {
	out := append([]dup.Cluster(nil), clusters...)
	for i := range out {
		occ := append([]dup.Occurrence(nil), out[i].Occurrences...)
		for j := range occ {
			occ[j].File = relFile(cwd, abs, occ[j].File)
		}
		out[i].Occurrences = occ
	}
	return out
}

func relSection(cwd string, abs map[string]string, sec models.Section) models.Section {
	sec.Models = relModels(cwd, abs, sec.Models)
	sec.Hotspots = relModels(cwd, abs, sec.Hotspots)
	return sec
}

func relModels(cwd string, abs map[string]string, list []models.Model) []models.Model {
	out := append([]models.Model(nil), list...)
	for i := range out {
		out[i].File = relFile(cwd, abs, out[i].File)
	}
	return out
}

func relFile(cwd string, abs map[string]string, stored string) string {
	target := stored
	if abs != nil {
		if found := abs[stored]; found != "" {
			target = found
		}
	}
	if !filepath.IsAbs(target) {
		return filepath.ToSlash(stored)
	}
	rel, err := filepath.Rel(cwd, target)
	if err != nil {
		return filepath.ToSlash(stored)
	}
	return filepath.ToSlash(rel)
}

func relDiags(cwd string, abs map[string]string, diags []string) []string {
	out := append([]string(nil), diags...)
	reps := diagReps(cwd, abs)
	for i := range out {
		out[i] = applyReps(out[i], reps)
	}
	return out
}

type pathRep struct{ from, to string }

func diagReps(cwd string, abs map[string]string) []pathRep {
	var reps []pathRep
	for stored := range abs {
		rel := relFile(cwd, abs, stored)
		reps = append(reps, pathRep{abs[stored], rel}, pathRep{stored, rel})
	}
	sort.Slice(reps, func(i, j int) bool { return len(reps[i].from) > len(reps[j].from) })
	return reps
}

func applyReps(text string, reps []pathRep) string {
	for _, r := range reps {
		text = swapPath(text, r.from, r.to)
	}
	return text
}

func swapPath(text, from, to string) string {
	if from == "" || from == to {
		return text
	}
	return strings.ReplaceAll(text, from, to)
}

func indexAbs(files []parsed) map[string]string {
	out := map[string]string{}
	for _, f := range files {
		if f.abs != "" {
			out[f.path] = f.abs
		}
	}
	return out
}

func internalFns(cwd, gitRoot string, fns []metric.Function) []metric.Function {
	out := append([]metric.Function(nil), fns...)
	for i := range out {
		out[i].File = internalPath(cwd, gitRoot, out[i].File)
	}
	return out
}

func internalModels(cwd, gitRoot string, list []models.Model) []models.Model {
	out := append([]models.Model(nil), list...)
	for i := range out {
		out[i].File = internalPath(cwd, gitRoot, out[i].File)
	}
	return out
}

func internalPath(cwd, gitRoot, shown string) string {
	if shown == "" || gitRoot == "" {
		return filepath.ToSlash(shown)
	}
	target := shown
	if !filepath.IsAbs(shown) {
		target = filepath.Join(cwd, shown)
	}
	rel, err := filepath.Rel(gitRoot, target)
	if err != nil {
		return filepath.ToSlash(shown)
	}
	return filepath.ToSlash(rel)
}

func userPath(given string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.ToSlash(given)
	}
	return config.Relative(cwd, given)
}
