package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ryangerardwilson/c3po/complexity/internal/agents"
	"github.com/ryangerardwilson/c3po/complexity/internal/config"
	"github.com/ryangerardwilson/c3po/complexity/internal/hook"
	"github.com/ryangerardwilson/c3po/complexity/internal/version"
)

const usageText = `complexity (cx) measures cognitive complexity and schema shape.

Usage:
  complexity [command] [path...] [flags]

Commands:
  check              Fail when a threshold is exceeded
  summary            Show the rollup and any real hotspots
  rank [metric] [n]  Rank functions by a metric and exit 0
  top                Same as: rank cognitive
  note [path...]     Report what is new or worse and exit 0
  budget [path...]   Show the room left in each function and exit 0
  accept             Record a score that may stay over a limit
  models             Score ORM relationship graphs
  schema             Score SQL tables and foreign keys
  diff <ref>         Analyze files changed since a git ref
  report             Write a text, JSON, SARIF, CSV, or HTML report
  baseline           Save scores for a later ratchet
  config             Print the merged config; --init writes the XDG settings.json
  status             Print enforcement status as JSON
  doctor             Check the config, AGENTS block, and hook
  hook               Read one agent hook event on stdin
  agents-block       Print the managed AGENTS.md block
  version
  help

Rank metrics: cognitive, nesting, lines, params, cyclomatic.
An integer after the metric caps the text list. The default cap is 20.

Flags:
  --max-cognitive N   --max-nesting N   --max-lines N   --max-params N
  --max-degree N      --max-depth N     --max-m2m N
  --diff REF          --ratchet         --baseline FILE
  --reason TEXT       --metric NAME
  --format text|json|sarif|csv|html     --output FILE
  --lang py,go,rust,ts                  --exclude GLOB
  --include-tests     --hotspots-only   --quiet   --strict-conventions
  --config FILE

Exit 0 is clean, 1 is a threshold violation, 2 is a fatal error.
note compares the given files with HEAD unless --diff is set.
accept <file> <function> --reason "..." writes .complexity-accept.json at the git root.
note, budget, rank, top, and summary exit 0 when a score is over a limit.
`

// Execute runs one CLI invocation.
func Execute(args []string, stdout, stderr io.Writer) int {
	if code, ok := earlyCmd(args, stdout); ok {
		return code
	}
	opt, err := parseArgs(args, stderr)
	if err != nil {
		return 2
	}
	if opt.cmd == "" {
		opt.cmd = defaultCommand()
	}
	return dispatch(opt, stdout, stderr)
}

func earlyCmd(args []string, stdout io.Writer) (int, bool) {
	if len(args) != 1 {
		return 0, false
	}
	if helpArg(args[0]) {
		fmt.Fprint(stdout, usageText)
		return 0, true
	}
	if versionArg(args[0]) {
		writeVersion(stdout)
		return 0, true
	}
	return 0, false
}

func helpArg(s string) bool {
	return s == "-h" || s == "--help" || s == "help"
}

func versionArg(s string) bool {
	return s == "version" || s == "--version"
}

func writeVersion(w io.Writer) {
	fmt.Fprintf(w, "complexity %s %s %s\n", version.Version, version.Commit, version.SourceHash)
}

func dispatch(opt options, stdout, stderr io.Writer) int {
	switch opt.cmd {
	case "help":
		fmt.Fprint(stdout, usageText)
		return 0
	case "version":
		writeVersion(stdout)
		return 0
	case "agents-block":
		fmt.Fprint(stdout, agents.Block)
		return 0
	case "hook":
		return hook.Run(os.Stdin, stdout)
	case "config":
		return runConfig(opt, stdout, stderr)
	case "status":
		return runStatus(opt, stdout, stderr)
	case "doctor":
		return runDoctor(stdout)
	case "accept":
		return runAccept(opt, stdout, stderr)
	default:
		return runAnalyze(opt, stdout, stderr)
	}
}

func defaultCommand() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "summary"
	}
	if config.Configured(cwd) {
		return "check"
	}
	return "summary"
}

func langsOf(spec string) map[string]bool {
	if strings.TrimSpace(spec) == "" {
		return nil
	}
	out := map[string]bool{}
	for _, part := range strings.Split(spec, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "py", "python":
			out["python"] = true
		case "go":
			out["go"] = true
		case "rs", "rust":
			out["rust"] = true
		case "ts", "tsx", "typescript", "js", "javascript":
			out["typescript"] = true
			out["javascript"] = true
		case "prisma":
			out["prisma"] = true
		case "sql":
			out["sql"] = true
		}
	}
	return out
}
