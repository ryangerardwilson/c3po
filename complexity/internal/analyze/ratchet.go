package analyze

import (
	"os"
	"path/filepath"

	"github.com/ryangerardwilson/c3po/complexity/internal/accept"
	"github.com/ryangerardwilson/c3po/complexity/internal/change"
	"github.com/ryangerardwilson/c3po/complexity/internal/discover"
	"github.com/ryangerardwilson/c3po/complexity/internal/dup"
	"github.com/ryangerardwilson/c3po/complexity/internal/gate"
	"github.com/ryangerardwilson/c3po/complexity/internal/linediff"
	"github.com/ryangerardwilson/c3po/complexity/internal/models"
	"github.com/ryangerardwilson/c3po/complexity/internal/parse"
)

// maxSiblings caps the extra files read to find a copied block.
const maxSiblings = 300

// changeSet says which files changed since ref, by git root. A root whose
// set is nil, or a run with an explicit --diff, counts every file.
type changeSet struct {
	all  bool
	sets map[string]map[string]bool
}

func changedSets(files []parsed, ref string, opt Options) changeSet {
	cs := changeSet{all: opt.Diff != "", sets: map[string]map[string]bool{}}
	if cs.all {
		return cs
	}
	for _, f := range files {
		if _, done := cs.sets[f.root]; done || f.root == "" {
			continue
		}
		set, err := changedFiles(f.root, ref)
		if err != nil {
			set = nil
		}
		cs.sets[f.root] = set
	}
	return cs
}

func (cs changeSet) has(f parsed) bool {
	if cs.all || f.root == "" {
		return true
	}
	set := cs.sets[f.root]
	return set == nil || set[f.path]
}

func refOf(opt Options) string {
	if opt.Diff != "" {
		return opt.Diff
	}
	return "HEAD"
}

func baseText(f parsed, ref string) ([]byte, bool) {
	if f.root == "" {
		return nil, false
	}
	return gitShow(f.root, ref, f.path)
}

// applyRatchet fails only new or worse scores in files changed since ref.
// A file nobody changed is its own baseline.
func applyRatchet(res *Result, opt Options, files []parsed) error {
	ref := refOf(opt)
	changed := changedSets(files, ref, opt)
	in := change.Input{Th: res.Thresholds, Maps: map[string]linediff.Map{}, Accepted: books(files)}
	var touched []parsed
	for _, f := range files {
		if !codeScore(f.lang) || !changed.has(f) {
			continue
		}
		touched = append(touched, f)
		in.Cur = append(in.Cur, f.fns...)
		if src, ok := baseText(f, ref); ok {
			fns, _ := parse.File(f.path, f.lang, src)
			in.Base = append(in.Base, fns...)
			in.Maps[f.path] = linediff.Lines(string(src), f.text)
		}
	}
	facts, bad := change.Build(in)
	res.Report = change.Report{Ref: ref, Changes: facts}
	res.Hotspots = bad
	ratchetModels(res, opt, files, ref, changed)
	if opt.Config.Duplication.Enabled {
		res.Report.Duplicates = introduced(res, touched, files, in.Maps, opt)
	}
	res.Violations = res.Report.Failing()
	return nil
}

// ratchetModels compares the model graph with the same graph at ref. An
// unchanged file contributes its current text, so a model that was already
// invalid there is not new.
func ratchetModels(res *Result, opt Options, files []parsed, ref string, changed changeSet) {
	var srcs []models.Source
	for _, f := range files {
		if !modelLang(f.lang, opt) {
			continue
		}
		text := f.text
		if changed.has(f) {
			src, ok := baseText(f, ref)
			if !ok {
				continue
			}
			text = string(src)
		}
		srcs = append(srcs, models.Source{Path: f.path, Language: f.lang, Text: text})
	}
	base := models.Scan(srcs, modelTh(opt.Config), modelConv(opt)).Models
	bad := gate.ModelRegressions(res.Models.Models, base, modelTh(opt.Config))
	res.Models.Hotspots = bad
	res.Report.Models = change.ModelFacts(bad, base)
}

func books(files []parsed) accept.Book {
	seen := map[string]bool{}
	var all []accept.Book
	for _, f := range files {
		if f.root == "" || seen[f.root] {
			continue
		}
		seen[f.root] = true
		if b, err := accept.Load(f.root); err == nil {
			all = append(all, b)
		}
	}
	return accept.Merge(all...)
}

// introduced finds changed lines that copy code already present in the
// analyzed files or beside a changed file.
func introduced(res *Result, touched, files []parsed, maps map[string]linediff.Map, opt Options) []change.Dup {
	fresh := map[string]dup.Fresh{}
	for _, f := range touched {
		fresh[f.path] = func(int) bool { return true }
		if m, ok := maps[f.path]; ok {
			fresh[f.path] = m.Fresh
		}
	}
	if len(fresh) == 0 {
		return nil
	}
	pool := append(append([]parsed(nil), files...), siblings(touched, files, opt)...)
	for _, f := range pool {
		res.abs[f.path] = f.abs
	}
	hits := dup.Introduced(fresh, dupFiles(pool), bodies(pool), opt.Config.Duplication.MinLines)
	return change.DupFacts(hits)
}

func bodies(files []parsed) map[string][]dup.Body {
	out := map[string][]dup.Body{}
	for _, f := range files {
		for _, fn := range f.fns {
			out[f.path] = append(out[f.path], dup.Body{fn.Line, fn.EndLine})
		}
	}
	return out
}

// siblings reads source files in the directories of changed files that the
// run did not already read.
func siblings(touched, files []parsed, opt Options) []parsed {
	have := map[string]bool{}
	for _, f := range files {
		have[f.abs] = true
	}
	var out []parsed
	for _, f := range touched {
		for _, abs := range dirFiles(filepath.Dir(f.abs)) {
			if len(out) >= maxSiblings {
				return out
			}
			if p, ok := sibling(abs, f.root, opt, have); ok {
				out = append(out, p)
			}
		}
	}
	return out
}

func dirFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.Type().IsRegular() {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out
}

func sibling(abs, root string, opt Options, have map[string]bool) (parsed, bool) {
	if have[abs] {
		return parsed{}, false
	}
	have[abs] = true
	rel := relTo(root, abs)
	lang, _ := discover.Language(rel)
	if !codeScore(lang) || discover.Skip(rel, opt.Config, opt.IncludeTests) {
		return parsed{}, false
	}
	buf, err := os.ReadFile(abs)
	if err != nil || len(buf) > 1_000_000 {
		return parsed{}, false
	}
	fns, _ := parse.File(rel, lang, buf)
	return parsed{path: rel, abs: abs, root: root, lang: lang, text: string(buf), fns: fns}, true
}

func relTo(root, abs string) string {
	if root == "" {
		return filepath.ToSlash(abs)
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}
