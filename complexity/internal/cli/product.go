package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ryangerardwilson/c3po/complexity/internal/accept"
	"github.com/ryangerardwilson/c3po/complexity/internal/analyze"
	"github.com/ryangerardwilson/c3po/complexity/internal/budget"
	"github.com/ryangerardwilson/c3po/complexity/internal/change"
	"github.com/ryangerardwilson/c3po/complexity/internal/config"
	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
)

func finish(opt options, cfg config.Config, res analyze.Result, stdout, stderr io.Writer) int {
	switch opt.cmd {
	case "baseline":
		return writeBase(opt, res, stdout, stderr)
	case "budget":
		return renderBudget(res, stdout, stderr)
	case "note":
		return renderNote(opt, cfg, res, stdout, stderr)
	default:
		return render(opt, cfg, res, stdout, stderr)
	}
}

func analyzeOptions(opt options, cfg config.Config) analyze.Options {
	diff := opt.diff
	if opt.cmd == "note" && diff == "" {
		diff = "HEAD"
	}
	return analyze.Options{
		Paths: pathsOr(opt.paths), Config: cfg, Diff: diff, Baseline: opt.baseline,
		Ratchet: ratchetOn(opt), IncludeTests: opt.includeTests,
		Langs: langsOf(opt.lang), ExtraExclude: opt.excludes, WantModels: wantModels(opt.cmd),
		SQLOnly: opt.cmd == "schema", ORMOnly: opt.cmd == "models", Strict: opt.strict,
	}
}

func ratchetOn(opt options) bool {
	return opt.ratchet || opt.baseline != "" || opt.cmd == "note"
}

func withChanges(opt options) bool {
	return opt.cmd == "note" || opt.ratchet || opt.baseline != ""
}

func ratchetPtr(opt options, res analyze.Result) *change.Report {
	if !withChanges(opt) {
		return nil
	}
	rep := res.Report
	return &rep
}

func renderBudget(res analyze.Result, stdout, stderr io.Writer) int {
	stored := res.Functions
	shown := res.ForDisplay()
	if len(stored) != len(shown.Functions) {
		fmt.Fprintln(stderr, "budget display mismatch")
		return 2
	}
	book, err := accept.Load(res.GitRoot)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	for _, e := range budget.Of(stored, shown.Functions, res.Thresholds, book) {
		fmt.Fprintln(stdout, e.Text())
	}
	return 0
}

func renderNote(opt options, cfg config.Config, res analyze.Result, stdout, stderr io.Writer) int {
	if pickFormat(opt, cfg) != "text" {
		return render(opt, cfg, res, stdout, stderr)
	}
	for _, line := range res.ForDisplay().Report.Lines() {
		fmt.Fprintln(stdout, line)
	}
	return 0
}

func runAccept(opt options, stdout, stderr io.Writer) int {
	if msg := acceptUsage(opt); msg != "" {
		fmt.Fprintln(stderr, msg)
		return 2
	}
	fns, res, err := acceptedFunctions(opt)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	return writeAcceptance(opt, res, fns, stdout, stderr)
}

func acceptUsage(opt options) string {
	if opt.reason == "" || len(opt.paths) < 2 {
		return `accept requires <file> <function> --reason "..."`
	}
	if opt.metric != "" && !knownMetric(opt.metric) {
		return "unknown metric " + opt.metric
	}
	return ""
}

func acceptedFunctions(opt options) ([]metric.Function, analyze.Result, error) {
	var res analyze.Result
	cwd, err := os.Getwd()
	if err != nil {
		return nil, res, err
	}
	cfg, err := config.Merge(cwd, opt.configPath)
	if err != nil {
		return nil, res, err
	}
	applyThresholds(&cfg, opt)
	res, err = analyze.Run(analyze.Options{
		Paths: []string{opt.paths[0]}, Config: cfg,
		IncludeTests: opt.includeTests, Langs: langsOf(opt.lang),
	})
	if err != nil {
		return nil, res, err
	}
	if res.GitRoot == "" {
		return nil, res, fmt.Errorf("accept records live at the git root")
	}
	fns := functionsNamed(res.Functions, opt.paths[1])
	if len(fns) == 0 {
		return nil, res, fmt.Errorf("no function %s", opt.paths[1])
	}
	return fns, res, nil
}

func functionsNamed(fns []metric.Function, name string) []metric.Function {
	var out []metric.Function
	for _, fn := range fns {
		if fn.Name == name {
			out = append(out, fn)
		}
	}
	return out
}

type acceptJob struct {
	book *accept.Book
	opt  options
	res  analyze.Result
	cwd  string
	now  string
	out  io.Writer
}

func writeAcceptance(opt options, res analyze.Result, fns []metric.Function, stdout, stderr io.Writer) int {
	book, err := accept.Load(res.GitRoot)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	cwd, _ := os.Getwd()
	job := acceptJob{
		book: &book, opt: opt, res: res, cwd: cwd,
		now: time.Now().UTC().Format(time.RFC3339), out: stdout,
	}
	n := 0
	for _, fn := range fns {
		n += recordFn(&job, fn)
	}
	if n == 0 {
		fmt.Fprintln(stderr, "no metric is over its limit")
		return 2
	}
	if err := accept.Save(res.GitRoot, book); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	return 0
}

func recordFn(job *acceptJob, fn metric.Function) int {
	names := namesFor(fn, job.res.Thresholds, job.opt.metric)
	for _, name := range names {
		score := metric.Value(fn, name)
		job.book.Put(accept.Record{
			File: fn.File, Function: fn.Name, Metric: name,
			Score: score, Reason: job.opt.reason, AcceptedAt: job.now,
		})
		fmt.Fprintf(job.out, "accepted %s %s %s %d\n", shownFile(job.cwd, job.res.GitRoot, fn.File), fn.Name, name, score)
	}
	return len(names)
}

func namesFor(fn metric.Function, th metric.Thresholds, only string) []string {
	if only != "" {
		return []string{only}
	}
	var names []string
	for _, name := range metric.Names {
		limit := metric.Limit(th, name)
		if limit > 0 && metric.Value(fn, name) > limit {
			names = append(names, name)
		}
	}
	return names
}

func knownMetric(name string) bool {
	for _, n := range metric.Names {
		if n == name {
			return true
		}
	}
	return false
}

func shownFile(cwd, root, file string) string {
	abs := file
	if root != "" && !filepath.IsAbs(file) {
		abs = filepath.Join(root, file)
	}
	return config.Relative(cwd, abs)
}
