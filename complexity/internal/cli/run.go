package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/ryangerardwilson/c3po/complexity/internal/agents"
	"github.com/ryangerardwilson/c3po/complexity/internal/analyze"
	"github.com/ryangerardwilson/c3po/complexity/internal/config"
	"github.com/ryangerardwilson/c3po/complexity/internal/discover"
	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
	"github.com/ryangerardwilson/c3po/complexity/internal/models"
	"github.com/ryangerardwilson/c3po/complexity/internal/report"
)

func runAnalyze(opt options, stdout, stderr io.Writer) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	cfg, err := config.Merge(cwd, opt.configPath)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	applyThresholds(&cfg, opt)
	res, err := analyze.Run(analyzeOptions(opt, cfg))
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	if opt.verbose {
		for _, d := range res.ForDisplay().Diagnostics {
			fmt.Fprintln(stderr, d)
		}
	}
	return finish(opt, cfg, res, stdout, stderr)
}

func writeBase(opt options, res analyze.Result, stdout, stderr io.Writer) int {
	dest := opt.output
	if dest == "" {
		dest = ".complexity-baseline.json"
	}
	shown := res.ForDisplay()
	if err := analyze.WriteBaseline(dest, shown.Functions, shown.Models.Models); err != nil {
		fmt.Fprintln(stderr, showErr(err, dest))
		return 2
	}
	cwd, _ := os.Getwd()
	fmt.Fprintf(stdout, "wrote %s\n", config.Relative(cwd, dest))
	return 0
}

func render(opt options, cfg config.Config, res analyze.Result, stdout, stderr io.Writer) int {
	res = res.ForDisplay()
	format := pickFormat(opt, cfg)
	code := 0
	if fails(opt.cmd) && res.Violations {
		code = 1
	}
	var sec *models.Section
	if res.HasModels {
		sec = &res.Models
	}
	doc := report.Build(report.Input{
		Functions: res.Functions, Hotspots: res.Hotspots, Clusters: res.Clusters,
		Thresholds: res.Thresholds, Models: sec, Failed: res.Violations,
		Diagnostics: res.Diagnostics, Ratchet: ratchetPtr(opt, res),
	})
	view := report.View{
		Format: format, Top: topOf(opt, cfg), HotspotsOnly: opt.hotspotsOnly,
		Quiet: opt.quiet, ShowExit: showExit(opt.cmd, format), ExitCode: code,
	}
	if opt.cmd == "rank" {
		view.RankBy = rankMetricOf(opt)
		doc.Ranked = report.Order(res.Functions, view.RankBy)
	}
	return writeReport(opt.output, stdout, stderr, doc, res.Functions, view)
}

func pickFormat(opt options, cfg config.Config) string {
	format := opt.format
	if format == "" {
		format = cfg.Output.Format
	}
	if reportHTML(opt) {
		format = "html"
	}
	if format == "" {
		return "text"
	}
	return format
}

func reportHTML(opt options) bool {
	return opt.cmd == "report" && opt.format == "" && !opt.set["format"] && !opt.set["f"]
}

func showExit(cmd, format string) bool {
	return cmd == "check" && format == "text"
}

func writeReport(path string, stdout, stderr io.Writer, doc report.Document, fns []metric.Function, view report.View) int {
	out, closer, err := dest(path, stdout)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	defer closer()
	if err := report.Write(out, doc, fns, view); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	return view.ExitCode
}

func showErr(err error, path string) string {
	if err == nil {
		return ""
	}
	if path == "" {
		return err.Error()
	}
	cwd, _ := os.Getwd()
	shown := config.Relative(cwd, path)
	if shown == "" || shown == path {
		return err.Error()
	}
	return strings.ReplaceAll(err.Error(), path, shown)
}

func dest(path string, stdout io.Writer) (io.Writer, func(), error) {
	if path == "" {
		return stdout, func() {}, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, func() {}, fmt.Errorf("%s", showErr(err, path))
	}
	return f, func() { f.Close() }, nil
}

func fails(cmd string) bool {
	switch cmd {
	case "check", "models", "schema", "report":
		return true
	default:
		return false
	}
}

func wantModels(cmd string) bool {
	switch cmd {
	case "summary", "top", "rank", "budget":
		return false
	default:
		return true
	}
}

func pathsOr(paths []string) []string {
	if len(paths) == 0 {
		return []string{"."}
	}
	return paths
}

func rankMetricOf(opt options) string {
	if opt.rankBy == "" {
		return "cognitive"
	}
	return opt.rankBy
}

func topOf(opt options, cfg config.Config) int {
	if opt.set["top"] {
		return opt.top
	}
	if cfg.Output.Top > 0 {
		return cfg.Output.Top
	}
	return 20
}

func applyThresholds(cfg *config.Config, opt options) {
	set := func(flag string, dst *int, val int) {
		if opt.set[flag] {
			*dst = val
		}
	}
	set("max-cognitive", &cfg.Thresholds.Cognitive, opt.maxCognitive)
	set("max-nesting", &cfg.Thresholds.Nesting, opt.maxNesting)
	set("max-lines", &cfg.Thresholds.Lines, opt.maxLines)
	set("max-params", &cfg.Thresholds.Params, opt.maxParams)
	set("max-cyclomatic", &cfg.Thresholds.Cyclomatic, opt.maxCyclomatic)
	set("max-degree", &cfg.Thresholds.Models.Degree, opt.maxDegree)
	set("max-depth", &cfg.Thresholds.Models.Depth, opt.maxDepth)
	set("max-m2m", &cfg.Thresholds.Models.M2M, opt.maxM2M)
	if opt.set["hotspots-only"] {
		cfg.Output.HotspotsOnly = true
	}
	if opt.strict {
		cfg.Conventions.Strict = true
	}
}

func runConfig(opt options, stdout, stderr io.Writer) int {
	cwd, _ := os.Getwd()
	if opt.init {
		path, err := config.GlobalPath()
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 2
		}
		if err := config.Ensure(path); err != nil {
			fmt.Fprintln(stderr, showErr(err, path))
			return 2
		}
		fmt.Fprintln(stdout, config.Relative(cwd, path))
		return 0
	}
	cfg, err := config.Merge(cwd, opt.configPath)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	applyThresholds(&cfg, opt)
	return encodeTOML(stdout, cfg)
}

func encodeTOML(w io.Writer, cfg config.Config) int {
	if err := toml.NewEncoder(w).Encode(cfg); err != nil {
		fmt.Fprintln(w, err.Error())
		return 2
	}
	return 0
}

type statusDoc struct {
	Enforced   bool                   `json:"enforced"`
	Mode       string                 `json:"mode"`
	Root       string                 `json:"root"`
	Config     string                 `json:"config"`
	Thresholds metric.Thresholds      `json:"thresholds"`
	Models     config.ModelThresholds `json:"models"`
}

func runStatus(opt options, stdout, stderr io.Writer) int {
	invoke, _ := os.Getwd()
	dir := invoke
	if len(opt.paths) > 0 {
		dir = opt.paths[0]
	}
	cfg, err := config.Merge(dir, opt.configPath)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	root := dir
	if git, ok := config.GitRoot(dir); ok {
		root = git
	}
	doc := statusDoc{
		Enforced: enforced(cfg, root), Mode: cfg.Enforcement.Mode,
		Root: config.Relative(invoke, root), Config: config.Relative(invoke, configFile(dir)),
		Thresholds: cfg.FuncThresholds(), Models: cfg.Thresholds.Models,
	}
	buf, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}
	fmt.Fprintln(stdout, string(buf))
	return 0
}

func enforced(cfg config.Config, root string) bool {
	switch cfg.Enforcement.Mode {
	case "off":
		return false
	case "on":
		return true
	default:
		return discover.HasCode(root, cfg)
	}
}

func configFile(cwd string) string {
	if path, err := config.GlobalPath(); err == nil {
		if _, err := os.Stat(path); err == nil {
			stop := cwd
			if root, ok := config.GitRoot(cwd); ok {
				stop = root
			}
			if project, err := config.FindProject(cwd, stop); err == nil && project != "" {
				return project
			}
			return path
		}
	}
	return ""
}

func runDoctor(stdout io.Writer) int {
	cwd, _ := os.Getwd()
	ok := true
	path, err := config.GlobalPath()
	if err != nil {
		fmt.Fprintln(stdout, "config: unavailable")
		ok = false
	} else if _, err := os.Stat(path); err != nil {
		fmt.Fprintf(stdout, "config: missing %s\n", config.Relative(cwd, path))
		ok = false
	} else {
		fmt.Fprintf(stdout, "config: %s\n", config.Relative(cwd, path))
	}
	if agentsPresent(cwd) {
		fmt.Fprintln(stdout, "agents: complexity block present")
	} else {
		fmt.Fprintln(stdout, "agents: complexity block missing")
		ok = false
	}
	if hookPresent() {
		fmt.Fprintln(stdout, "hook: complexity hook present")
	} else {
		fmt.Fprintln(stdout, "hook: complexity hook missing")
		ok = false
	}
	if !ok {
		return 1
	}
	return 0
}

func agentsPresent(cwd string) bool {
	root := cwd
	if git, ok := config.GitRoot(cwd); ok {
		root = git
	}
	buf, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		return false
	}
	return strings.Contains(string(buf), "<!-- complexity:begin -->") && strings.Contains(string(buf), agents.Recheck)
}

func hookPresent() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	buf, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		return false
	}
	return strings.Contains(string(buf), "complexity hook")
}
