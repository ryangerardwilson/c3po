package cli

import (
	"fmt"
	"io"
	"strings"
)

type options struct {
	cmd           string
	paths         []string
	diff          string
	baseline      string
	format        string
	output        string
	lang          string
	configPath    string
	excludes      []string
	includeTests  bool
	hotspotsOnly  bool
	quiet         bool
	verbose       bool
	ratchet       bool
	strict        bool
	init          bool
	maxCognitive  int
	maxNesting    int
	maxLines      int
	maxParams     int
	maxCyclomatic int
	maxDegree     int
	maxDepth      int
	maxM2M        int
	top           int
	rankBy        string
	reason        string
	metric        string
	set           map[string]bool
}

func parseArgs(args []string, stderr io.Writer) (options, error) {
	opt := options{
		maxCognitive: -1, maxNesting: -1, maxLines: -1, maxParams: -1, maxCyclomatic: -1,
		maxDegree: -1, maxDepth: -1, maxM2M: -1, top: -1, set: map[string]bool{},
	}
	rest, err := takeCommand(&opt, args)
	if err != nil {
		return opt, err
	}
	for i := 0; i < len(rest); i++ {
		a := rest[i]
		if a == "--" {
			opt.paths = append(opt.paths, rest[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") {
			opt.paths = append(opt.paths, a)
			continue
		}
		name, val, consumed, err := oneFlag(a, rest, i)
		if err != nil {
			return opt, err
		}
		i += consumed
		if err := apply(&opt, name, val); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return opt, err
		}
	}
	return opt, nil
}

func takeCommand(opt *options, args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, nil
	}
	if args[0] == "diff" {
		return takeDiff(opt, args)
	}
	if args[0] == "rank" || args[0] == "top" {
		return takeRank(opt, args[1:])
	}
	if command(args[0]) {
		opt.cmd = args[0]
		return args[1:], nil
	}
	return args, nil
}

func takeDiff(opt *options, args []string) ([]string, error) {
	if len(args) < 2 || strings.HasPrefix(args[1], "-") {
		return nil, fmt.Errorf("diff requires a git ref")
	}
	opt.cmd = "check"
	opt.diff = args[1]
	opt.set["diff"] = true
	return args[2:], nil
}

func takeRank(opt *options, args []string) ([]string, error) {
	opt.cmd = "rank"
	opt.rankBy = "cognitive"
	if len(args) == 0 {
		return nil, nil
	}
	if metric, ok := canonicalMetric(args[0]); ok {
		opt.rankBy = metric
		args = args[1:]
	}
	return takeRankLimit(opt, args)
}

func takeRankLimit(opt *options, args []string) ([]string, error) {
	if len(args) == 0 || !isDigits(args[0]) {
		return args, nil
	}
	if err := setInt(&opt.top, args[0]); err != nil {
		return nil, err
	}
	opt.set["top"] = true
	return args[1:], nil
}

func canonicalMetric(s string) (string, bool) {
	switch s {
	case "cognitive", "cog":
		return "cognitive", true
	case "nesting", "nest":
		return "nesting", true
	case "lines", "length":
		return "lines", true
	case "params", "parameters":
		return "params", true
	case "cyclomatic", "cyclo":
		return "cyclomatic", true
	default:
		return "", false
	}
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func command(s string) bool {
	switch s {
	case "check", "summary", "rank", "top", "note", "budget", "accept", "models", "schema", "report", "baseline", "config",
		"status", "doctor", "hook", "version", "help", "agents-block":
		return true
	default:
		return false
	}
}

func oneFlag(a string, args []string, i int) (string, string, int, error) {
	name := strings.TrimLeft(a, "-")
	if eq := strings.IndexByte(name, '='); eq >= 0 {
		return name[:eq], name[eq+1:], 0, nil
	}
	if !takesValue(name) {
		return name, "true", 0, nil
	}
	if i+1 >= len(args) {
		return "", "", 0, fmt.Errorf("flag --%s needs a value", name)
	}
	return name, args[i+1], 1, nil
}

func takesValue(name string) bool {
	switch name {
	case "p", "path", "l", "lang", "exclude", "config", "max-cognitive", "max-nesting",
		"max-lines", "max-params", "max-cyclomatic", "max-degree", "max-depth", "max-m2m",
		"f", "format", "o", "output", "top", "baseline", "diff", "reason", "metric":
		return true
	default:
		return false
	}
}

func apply(opt *options, name, val string) error {
	opt.set[name] = true
	if done, err := applyIO(opt, name, val); done {
		return err
	}
	if applyMeta(opt, name, val) {
		return nil
	}
	if done, err := applyNums(opt, name, val); done {
		return err
	}
	return applyFlags(opt, name, val)
}

func applyIO(opt *options, name, val string) (bool, error) {
	switch name {
	case "p", "path":
		opt.paths = append(opt.paths, val)
	case "l", "lang":
		opt.lang = val
	case "exclude":
		opt.excludes = append(opt.excludes, val)
	case "config":
		opt.configPath = val
	case "f", "format":
		opt.format = val
	case "o", "output":
		opt.output = val
	case "baseline":
		opt.baseline = val
	case "diff":
		opt.diff = val
	default:
		return false, nil
	}
	return true, nil
}

func applyNums(opt *options, name, val string) (bool, error) {
	if done, err := applyCount(opt, name, val); done {
		return true, err
	}
	return applyModelFlag(opt, name, val)
}

func applyCount(opt *options, name, val string) (bool, error) {
	switch name {
	case "top":
		return true, setInt(&opt.top, val)
	case "max-cognitive":
		return true, setInt(&opt.maxCognitive, val)
	case "max-nesting":
		return true, setInt(&opt.maxNesting, val)
	case "max-lines":
		return true, setInt(&opt.maxLines, val)
	case "max-params":
		return true, setInt(&opt.maxParams, val)
	default:
		return false, nil
	}
}

func applyModelFlag(opt *options, name, val string) (bool, error) {
	switch name {
	case "max-cyclomatic":
		return true, setInt(&opt.maxCyclomatic, val)
	case "max-degree":
		return true, setInt(&opt.maxDegree, val)
	case "max-depth":
		return true, setInt(&opt.maxDepth, val)
	case "max-m2m":
		return true, setInt(&opt.maxM2M, val)
	default:
		return false, nil
	}
}

func applyFlags(opt *options, name, val string) error {
	if applyRunFlags(opt, name) || applyExtraFlags(opt, name, val) {
		return nil
	}
	return fmt.Errorf("unknown flag --%s", name)
}

func applyRunFlags(opt *options, name string) bool {
	switch name {
	case "include-tests":
		opt.includeTests = true
	case "hotspots-only":
		opt.hotspotsOnly = true
	case "quiet":
		opt.quiet = true
	case "ratchet":
		opt.ratchet = true
	case "strict-conventions":
		opt.strict = true
	default:
		return false
	}
	return true
}

func applyExtraFlags(opt *options, name, val string) bool {
	switch name {
	case "v", "verbose":
		opt.verbose = true
	case "no-color", "check-conventions", "suggest-fixes":
		return true
	case "init":
		opt.init = true
	case "json":
		opt.format = "json"
	case "version":
		opt.cmd = "version"
	case "help", "h":
		opt.cmd = "help"
	default:
		_ = val
		return false
	}
	return true
}

func setInt(dst *int, val string) error {
	n := 0
	for _, c := range val {
		if c < '0' || c > '9' {
			return fmt.Errorf("expected an integer, got %s", val)
		}
		n = n*10 + int(c-'0')
	}
	*dst = n
	return nil
}

func applyMeta(opt *options, name, val string) bool {
	switch name {
	case "reason":
		opt.reason = val
	case "metric":
		opt.metric = metricName(val)
	default:
		return false
	}
	return true
}

func metricName(val string) string {
	if m, ok := canonicalMetric(val); ok {
		return m
	}
	return val
}
