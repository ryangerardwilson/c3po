package analyze

import (
	"os"
	"runtime"
	"sync"

	"github.com/ryangerardwilson/c3po/complexity/internal/change"
	"github.com/ryangerardwilson/c3po/complexity/internal/config"
	"github.com/ryangerardwilson/c3po/complexity/internal/discover"
	"github.com/ryangerardwilson/c3po/complexity/internal/dup"
	"github.com/ryangerardwilson/c3po/complexity/internal/gate"
	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
	"github.com/ryangerardwilson/c3po/complexity/internal/models"
	"github.com/ryangerardwilson/c3po/complexity/internal/parse"
)

// Options selects what a run measures.
type Options struct {
	Paths        []string
	Config       config.Config
	Diff         string
	Baseline     string
	Ratchet      bool
	IncludeTests bool
	Langs        map[string]bool
	ExtraExclude []string
	WantModels   bool
	SQLOnly      bool
	ORMOnly      bool
	Strict       bool
}

// Result is one analysis.
type Result struct {
	Functions   []metric.Function
	Hotspots    []metric.Function
	Clusters    []dup.Cluster
	Models      models.Section
	HasModels   bool
	Diagnostics []string
	Violations  bool
	Thresholds  metric.Thresholds
	GitRoot     string
	// Report states what is new or worse when the run compares a baseline.
	Report change.Report
	// Files lists the source files whose functions were scored.
	Files []string
	abs   map[string]string
}

type parsed struct {
	path, abs, root, lang, text string
	fns                         []metric.Function
}

// Run walks, scores, and applies the ratchet when requested.
func Run(opt Options) (Result, error) {
	if len(opt.Paths) == 0 {
		opt.Paths = []string{"."}
	}
	th := opt.Config.FuncThresholds()
	res := Result{Thresholds: th}
	parsed, err := gather(&res, opt)
	if err != nil {
		return res, err
	}
	scoreRun(&res, parsed, th, opt)
	if err := applyGate(&res, opt, parsed); err != nil {
		return res, err
	}
	return res, nil
}

func gather(res *Result, opt Options) ([]parsed, error) {
	var parsedFiles []parsed
	for _, p := range opt.Paths {
		files, root, err := collect(p, opt)
		if err != nil {
			return nil, err
		}
		if root != "" {
			res.GitRoot = root
		}
		got, diags := readAll(files)
		for i := range got {
			got[i].root = root
		}
		res.Diagnostics = append(res.Diagnostics, diags...)
		parsedFiles = append(parsedFiles, got...)
	}
	return parsedFiles, nil
}

func scoreRun(res *Result, parsedFiles []parsed, th metric.Thresholds, opt Options) {
	var modelSrc []models.Source
	for _, f := range parsedFiles {
		res.Functions = append(res.Functions, f.fns...)
		if codeScore(f.lang) {
			res.Files = append(res.Files, f.path)
		}
		if opt.WantModels && modelLang(f.lang, opt) {
			modelSrc = append(modelSrc, models.Source{Path: f.path, Language: f.lang, Text: f.text})
		}
	}
	for i := range res.Functions {
		res.Functions[i].Mark(th)
	}
	res.Hotspots = violators(res.Functions)
	if opt.Config.Duplication.Enabled && len(parsedFiles) <= 4000 {
		res.Clusters = dup.Find(dupFiles(parsedFiles), opt.Config.Duplication.MinLines)
	}
	if opt.WantModels {
		res.HasModels = true
		res.Models = models.Scan(modelSrc, modelTh(opt.Config), modelConv(opt))
	}
	res.abs = indexAbs(parsedFiles)
}

func applyGate(res *Result, opt Options, files []parsed) error {
	if opt.Baseline != "" {
		return applyBaseline(res, opt)
	}
	if !opt.Ratchet {
		return applyAbsolute(res, opt)
	}
	return applyRatchet(res, opt, files)
}

func applyBaseline(res *Result, opt Options) error {
	baseFns, baseModels, err := loadBaseline(opt.Baseline, res.GitRoot)
	if err != nil {
		return err
	}
	facts, hot := change.Build(change.Input{Cur: res.Functions, Base: baseFns, Th: res.Thresholds})
	res.Hotspots = hot
	bad := gate.ModelRegressions(res.Models.Models, baseModels, modelTh(opt.Config))
	res.Report = change.Report{Ref: userPath(opt.Baseline), Changes: facts, Models: change.ModelFacts(bad, baseModels)}
	res.Violations = res.Report.Failing()
	if len(bad) > 0 {
		res.Models.Hotspots = bad
	}
	return nil
}

func applyAbsolute(res *Result, opt Options) error {
	strict := opt.Strict || opt.Config.Conventions.Strict
	res.Violations = len(res.Hotspots) > 0 || modelHard(res.Models, strict)
	return nil
}


func modelHard(sec models.Section, strict bool) bool {
	for _, m := range sec.Models {
		if models.Hard(m, strict) {
			return true
		}
	}
	return false
}

func violators(fns []metric.Function) []metric.Function {
	var out []metric.Function
	for _, fn := range fns {
		if len(fn.Violations) > 0 {
			out = append(out, fn)
		}
	}
	return out
}

func codeScore(lang string) bool {
	switch lang {
	case "go", "python", "rust", "typescript", "javascript":
		return true
	default:
		return false
	}
}

func modelLang(lang string, opt Options) bool {
	if opt.SQLOnly {
		return lang == "sql"
	}
	if opt.ORMOnly && lang == "sql" {
		return false
	}
	switch lang {
	case "prisma", "sql", "python", "go", "typescript", "javascript", "rust":
		return true
	default:
		return false
	}
}

func modelTh(cfg config.Config) models.Thresholds {
	return models.Thresholds{Degree: cfg.Thresholds.Models.Degree, Depth: cfg.Thresholds.Models.Depth, M2M: cfg.Thresholds.Models.M2M}
}

func modelConv(opt Options) models.Conventions {
	c := opt.Config.Conventions
	return models.Conventions{
		Naming: c.Naming, ColumnOrder: c.ColumnOrder, DataColumnName: c.DataColumnName,
		LogSuffix: c.LogSuffix, RequireFeatureMapping: c.RequireFeatureMapping,
		Strict: c.Strict || opt.Strict,
	}
}

func dupFiles(files []parsed) []dup.File {
	var out []dup.File
	for _, f := range files {
		if codeScore(f.lang) {
			out = append(out, dup.File{Path: f.path, Src: f.text, Lang: f.lang})
		}
	}
	return out
}

func readAll(files []discover.File) ([]parsed, []string) {
	out := make([]parsed, len(files))
	diags := make([][]string, len(files))
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	var wg sync.WaitGroup
	for i, f := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, f discover.File) {
			defer wg.Done()
			defer func() { <-sem }()
			buf, err := os.ReadFile(f.Abs)
			if err != nil {
				out[i] = parsed{path: f.Path, abs: f.Abs}
				diags[i] = []string{f.Path + ": " + err.Error()}
				return
			}
			out[i] = parsed{path: f.Path, abs: f.Abs, lang: f.Language, text: string(buf)}
			if codeScore(f.Language) {
				fns, d := parse.File(f.Path, f.Language, buf)
				out[i].fns = fns
				diags[i] = d
			}
		}(i, f)
	}
	wg.Wait()
	var flat []string
	var got []parsed
	for i := range out {
		flat = append(flat, diags[i]...)
		if out[i].path != "" {
			got = append(got, out[i])
		}
	}
	return got, flat
}
