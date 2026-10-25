package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ryangerardwilson/c3po/ast/internal/agents"
	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/doctor"
	"github.com/ryangerardwilson/c3po/ast/internal/enforce"
	"github.com/ryangerardwilson/c3po/ast/internal/execgen"
	"github.com/ryangerardwilson/c3po/ast/internal/files"
	"github.com/ryangerardwilson/c3po/ast/internal/graph"
	"github.com/ryangerardwilson/c3po/ast/internal/hook"
	"github.com/ryangerardwilson/c3po/ast/internal/intel"
	"github.com/ryangerardwilson/c3po/ast/internal/measure"
	"github.com/ryangerardwilson/c3po/ast/internal/packages"
	"github.com/ryangerardwilson/c3po/ast/internal/root"
	"github.com/ryangerardwilson/c3po/ast/internal/trace"
	"github.com/ryangerardwilson/c3po/ast/internal/version"
)

const (
	ExitOK    = 0
	ExitFail  = 1
	ExitUsage = 2
)

type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

type Runner struct {
	Out          io.Writer
	Err          io.Writer
	In           io.Reader
	Cwd          string
	ConfigPath   string
	LoadConfig   func() (config.Config, string, error)
	RootOverride string
	Refresh      bool
}

func NewRunner(out, errOut io.Writer) Runner {
	return Runner{Out: out, Err: errOut, In: os.Stdin}
}

func (r Runner) Run(args []string) int {
	args, rootFlag, err := stripRootFlag(args)
	if err != nil {
		return r.usage(err.Error())
	}
	if rootFlag != "" {
		r.RootOverride = rootFlag
	}
	if len(args) == 0 {
		WriteHelp(r.Out)
		return ExitOK
	}
	cmd := args[0]
	rest := args[1:]
	rest, rootFlag2, err := stripRootFlag(rest)
	if err != nil {
		return r.usage(err.Error())
	}
	if rootFlag2 != "" {
		r.RootOverride = rootFlag2
	}
	// global --refresh before/after verb for status/measure
	rest, refresh, err := stripBoolFlag(rest, "--refresh")
	if err != nil {
		return r.usage(err.Error())
	}
	r.Refresh = refresh

	if len(rest) == 1 && (rest[0] == "--help" || rest[0] == "-h") && cmd != "search" {
		WriteCommandHelp(r.Out, cmd)
		return ExitOK
	}
	switch cmd {
	case "help", "-h", "--help":
		if len(rest) == 1 {
			WriteCommandHelp(r.Out, rest[0])
			return ExitOK
		}
		if len(rest) != 0 {
			return r.usage("Use: c3po ast help [command]")
		}
		WriteHelp(r.Out)
		return ExitOK
	case "version", "--version", "-v":
		err = r.cmdVersion(rest)
	case "config":
		err = r.cmdConfig(rest)
	case "status":
		err = r.cmdStatus(rest)
	case "measure":
		err = r.cmdMeasure(rest)
	case "map":
		err = r.cmdMap(rest)
	case "show":
		err = r.cmdShow(rest)
	case "search":
		err = r.cmdSearch(rest)
	case "callers":
		err = r.cmdTrace("callers", rest)
	case "callees":
		err = r.cmdTrace("callees", rest)
	case "impact":
		err = r.cmdTrace("impact", rest)
	case "explain":
		err = r.cmdExplain(rest)
	case "hash":
		err = r.cmdHash(rest)
	case "read":
		err = r.cmdRead(rest)
	case "create":
		err = r.cmdCreate(rest)
	case "append":
		err = r.cmdAppend(rest)
	case "patch":
		err = r.cmdPatch(rest)
	case "apply":
		err = r.cmdApply(rest)
	case "delete":
		err = r.cmdDelete(rest)
	case "move":
		err = r.cmdMove(rest)
	case "exec":
		err = r.cmdExec(rest)
	case "doctor":
		err = r.cmdDoctor(rest)
	case "agents-block":
		err = r.cmdAgentsBlock(rest)
	case "hook":
		err = r.cmdHook(rest)
	case "packages":
		err = r.cmdPackages(rest)
	case "affected":
		err = r.cmdAffected(rest)
	case "graph":
		err = r.cmdGraph(rest)
	case "neighbors":
		err = r.cmdNeighbors(rest)
	case "cut":
		err = r.cmdCut(rest)
	default:
		return r.usage(fmt.Sprintf("unknown command %q; see rgw-ast help", cmd))
	}
	if err == nil {
		return ExitOK
	}
	var ue usageError
	if errors.As(err, &ue) {
		_, _ = fmt.Fprintln(r.Err, "Error:", ue.Error())
		return ExitUsage
	}
	_, _ = fmt.Fprintln(r.Err, "Error:", err.Error())
	return ExitFail
}

func (r Runner) usage(msg string) int {
	_, _ = fmt.Fprintln(r.Err, "Error:", msg)
	return ExitUsage
}

func stripRootFlag(args []string) (rest []string, root string, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--root" {
			if i+1 >= len(args) {
				return nil, "", fmt.Errorf("--root requires a path")
			}
			if root != "" {
				return nil, "", fmt.Errorf("duplicate --root")
			}
			root = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(a, "--root=") {
			if root != "" {
				return nil, "", fmt.Errorf("duplicate --root")
			}
			root = strings.TrimPrefix(a, "--root=")
			continue
		}
		rest = append(rest, a)
	}
	return rest, root, nil
}

func stripBoolFlag(args []string, name string) ([]string, bool, error) {
	found := false
	var rest []string
	for _, a := range args {
		if a == name {
			if found {
				return nil, false, fmt.Errorf("duplicate %s", name)
			}
			found = true
			continue
		}
		rest = append(rest, a)
	}
	return rest, found, nil
}

func (r Runner) load() (config.Config, string, error) {
	if r.LoadConfig != nil {
		return r.LoadConfig()
	}
	if r.ConfigPath != "" {
		if err := config.Ensure(r.ConfigPath); err != nil {
			return config.Config{}, "", err
		}
		cfg, err := config.Read(r.ConfigPath)
		return cfg, r.ConfigPath, err
	}
	return config.Load()
}

func (r Runner) workspace() (cfg config.Config, cfgPath, wsRoot string, err error) {
	cfg, cfgPath, err = r.load()
	if err != nil {
		return
	}
	if r.RootOverride != "" {
		wsRoot, err = filepath.Abs(r.RootOverride)
		if err != nil {
			return
		}
		st, statErr := os.Stat(wsRoot)
		if statErr != nil {
			err = statErr
			return
		}
		if !st.IsDir() {
			err = fmt.Errorf("--root is not a directory: %s", wsRoot)
			return
		}
		return
	}
	cwd := r.Cwd
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			return
		}
	}
	wsRoot, err = root.Resolve(cwd)
	return
}

// workspaceFor is workspace for a command naming path. Without --root, an
// absolute path outside the resolved root resolves its own root, as the hook
// does, so one session can work across repositories.
func (r Runner) workspaceFor(path string) (cfg config.Config, cfgPath, wsRoot string, err error) {
	cfg, cfgPath, wsRoot, err = r.workspace()
	if err != nil || r.RootOverride != "" || !filepath.IsAbs(path) {
		return
	}
	clean := filepath.Clean(path)
	if rel, relErr := filepath.Rel(wsRoot, clean); relErr == nil && rel != ".." && !strings.HasPrefix(rel, "../") {
		return
	}
	dir := clean
	if st, statErr := os.Stat(clean); statErr != nil || !st.IsDir() {
		dir = filepath.Dir(clean)
	}
	for {
		if _, statErr := os.Stat(dir); statErr == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	wsRoot, err = root.Resolve(dir)
	return
}

func (r Runner) measureRoot(cfg config.Config, wsRoot string) (measure.Result, error) {
	return measure.CountCachedOpts(wsRoot, cfg, measure.CountOpts{Refresh: r.Refresh})
}

// measureFresh forces a recount for mutation/hook-adjacent enforcement.
func (r Runner) measureFresh(cfg config.Config, wsRoot string) (measure.Result, error) {
	return measure.CountCachedOpts(wsRoot, cfg, measure.CountOpts{Refresh: true})
}

func (r Runner) cmdVersion(args []string) error {
	asJSON, rest, err := takeJSONFlag(args)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return usageError{"Use: c3po ast version [--json]"}
	}
	if asJSON {
		return writeJSON(r.Out, map[string]any{
			"version":     version.Version,
			"commit":      version.Commit,
			"source_hash": version.SourceHash,
			"dirty":       version.Dirty(),
			"build_time":  version.BuildTime,
			"string":      version.String(),
		})
	}
	_, _ = fmt.Fprintln(r.Out, version.String())
	return nil
}

func (r Runner) cmdConfig(args []string) error {
	if len(args) != 0 {
		return usageError{"Use: c3po ast config"}
	}
	_, path, err := r.load()
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(r.Out, abs)
	return nil
}

type statusOut struct {
	Root               string `json:"root"`
	LOC                int    `json:"loc"`
	ProseLOC           int    `json:"prose_loc"`
	FileCount          int    `json:"file_count"`
	ThresholdLOC       int    `json:"threshold_loc"`
	Mode               string `json:"mode"`
	Enforced           bool   `json:"enforced"`
	Config             string `json:"config"`
	CacheHit           bool   `json:"cache_hit"`
	CacheAgeMs         int64  `json:"cache_age_ms"`
	NestedReposSkipped int    `json:"nested_repos_skipped"`
	GitIgnoreActive    bool   `json:"gitignore_active"`
	Branch             string `json:"branch"`
	Head               string `json:"head"`
	Upstream           string `json:"upstream"`
	Dirty              bool   `json:"dirty"`
	Detached           bool   `json:"detached"`
	Behind             int    `json:"behind"`
	MainWorktree       string `json:"main_worktree"`
}

func (r Runner) cmdStatus(args []string) error {
	asJSON, rest, err := takeJSONFlag(args)
	if err != nil {
		return err
	}
	rest, refresh, err := stripBoolFlag(rest, "--refresh")
	if err != nil {
		return err
	}
	if refresh {
		r.Refresh = true
	}
	if len(rest) != 0 {
		return usageError{"Use: c3po ast status [--json] [--refresh]"}
	}
	cfg, cfgPath, wsRoot, err := r.workspace()
	if err != nil {
		return err
	}
	m, err := r.measureRoot(cfg, wsRoot)
	if err != nil {
		return err
	}
	d := enforce.Decide(cfg, m.LOC)
	git := worktreeIdentity(wsRoot)
	out := statusOut{
		Root: m.Root, LOC: m.LOC, ProseLOC: m.ProseLOC, FileCount: m.FileCount,
		ThresholdLOC: d.ThresholdLOC, Mode: d.Mode, Enforced: d.Enforced,
		Config: cfgPath, CacheHit: m.CacheHit, CacheAgeMs: m.CacheAgeMs,
		NestedReposSkipped: m.NestedReposSkipped, GitIgnoreActive: m.GitIgnoreActive,
		Branch: git.branch, Head: git.head, Upstream: git.upstream, Dirty: git.dirty,
		Detached: git.detached, Behind: git.behind, MainWorktree: git.main,
	}
	if asJSON {
		return writeJSON(r.Out, out)
	}
	_, _ = fmt.Fprintf(r.Out, "root: %s\nloc: %d\nprose_loc: %d\nfile_count: %d\nthreshold_loc: %d\nmode: %s\nenforced: %v\nconfig: %s\ncache_hit: %v\ncache_age_ms: %d\nnested_repos_skipped: %d\ngitignore_active: %v\nbranch: %s\nhead: %s\nupstream: %s\ndirty: %v\ndetached: %v\nbehind: %d\nmain_worktree: %s\n",
		out.Root, out.LOC, out.ProseLOC, out.FileCount, out.ThresholdLOC, out.Mode, out.Enforced, out.Config, out.CacheHit, out.CacheAgeMs, out.NestedReposSkipped, out.GitIgnoreActive,
		out.Branch, out.Head, out.Upstream, out.Dirty, out.Detached, out.Behind, out.MainWorktree)
	return nil
}

type gitIdentity struct {
	branch, head, upstream, main string
	dirty, detached              bool
	behind                       int
}

func worktreeIdentity(root string) gitIdentity {
	var g gitIdentity
	g.branch = gitOut(root, "rev-parse", "--abbrev-ref", "HEAD")
	g.head = gitOut(root, "rev-parse", "HEAD")
	g.detached = g.branch == "HEAD"
	g.upstream = gitOut(root, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	porc := gitOut(root, "status", "--porcelain")
	g.dirty = porc != ""
	g.main = gitOut(root, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if g.main != "" {
		g.main = filepath.Dir(g.main)
	}
	if g.upstream != "" {
		if n := gitOut(root, "rev-list", "--count", "HEAD.."+g.upstream); n != "" {
			fmt.Sscan(n, &g.behind)
		}
	}
	return g
}

func gitOut(root string, args ...string) string {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (r Runner) cmdMeasure(args []string) error {
	asJSON, rest, err := takeJSONFlag(args)
	if err != nil {
		return err
	}
	rest, refresh, err := stripBoolFlag(rest, "--refresh")
	if err != nil {
		return err
	}
	if refresh {
		r.Refresh = true
	}
	if len(rest) != 0 {
		return usageError{"Use: c3po ast measure [--json] [--refresh]"}
	}
	cfg, _, wsRoot, err := r.workspace()
	if err != nil {
		return err
	}
	m, err := r.measureRoot(cfg, wsRoot)
	if err != nil {
		return err
	}
	if asJSON {
		return writeJSON(r.Out, map[string]any{
			"root": m.Root, "loc": m.LOC, "prose_loc": m.ProseLOC, "file_count": m.FileCount,
			"cache_hit": m.CacheHit, "nested_repos_skipped": m.NestedReposSkipped,
			"gitignore_active": m.GitIgnoreActive,
		})
	}
	_, _ = fmt.Fprintf(r.Out, "root: %s\nloc: %d\nprose_loc: %d\nfile_count: %d\n", m.Root, m.LOC, m.ProseLOC, m.FileCount)
	return nil
}

func (r Runner) cmdMap(args []string) error {
	asJSON := false
	offset, limit, depth := 0, 0, -1
	path := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			asJSON = true
		case a == "--offset" || a == "--limit" || a == "--depth":
			if i+1 >= len(args) {
				return usageError{a + " requires a value"}
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n < 0 {
				return usageError{a + " requires a non-negative integer"}
			}
			switch a {
			case "--offset":
				offset = n
			case "--limit":
				limit = n
			case "--depth":
				depth = n
			}
			i++
		case strings.HasPrefix(a, "--offset=") || strings.HasPrefix(a, "--limit=") || strings.HasPrefix(a, "--depth="):
			parts := strings.SplitN(a, "=", 2)
			n, err := strconv.Atoi(parts[1])
			if err != nil || n < 0 {
				return usageError{parts[0] + " requires a non-negative integer"}
			}
			switch parts[0] {
			case "--offset":
				offset = n
			case "--limit":
				limit = n
			case "--depth":
				depth = n
			}
		default:
			if strings.HasPrefix(a, "-") || path != "" {
				return usageError{"Use: c3po ast map [--json] [--limit N] [--offset N] [--depth N] [path]"}
			}
			path = a
		}
	}
	cfg, _, wsRoot, err := r.workspaceFor(path)
	if err != nil {
		return err
	}
	if limit <= 0 {
		limit = cfg.MaxMapEntries
	}
	res, err := intel.MapQuery(wsRoot, path, cfg, intel.MapOptions{Limit: limit, Offset: offset, Depth: depth})
	if err != nil {
		return err
	}
	if asJSON {
		return writeJSON(r.Out, res)
	}
	r.printMap(res)
	return nil
}

func (r Runner) printMap(res intel.MapResult) {
	for _, e := range res.Entries {
		_, _ = fmt.Fprintf(r.Out, "%s:%d %s\n", e.Path, e.Line, e.Label)
	}
	if res.Truncated {
		next := 0
		if res.NextOffset != nil {
			next = *res.NextOffset
		}
		_, _ = fmt.Fprintf(r.Out, "truncated: true next_offset: %d\n", next)
	}
}

func (r Runner) cmdShow(args []string) error {
	if len(args) != 1 {
		return usageError{"Use: c3po ast show <symbol|path:symbol|path>"}
	}
	cfg, _, wsRoot, err := r.workspaceFor(args[0])
	if err != nil {
		return err
	}
	// A bare path is not a symbol: outline it instead of failing.
	if !strings.Contains(args[0], ":") {
		p := args[0]
		if !filepath.IsAbs(p) {
			p = filepath.Join(wsRoot, p)
		}
		if _, statErr := os.Stat(p); statErr == nil {
			res, err := intel.MapQuery(wsRoot, p, cfg, intel.MapOptions{Limit: cfg.MaxMapEntries, Depth: -1})
			if err != nil {
				return err
			}
			r.printMap(res)
			return nil
		}
	}
	body, err := intel.Show(wsRoot, args[0], cfg)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(r.Out, body)
	return nil
}

func (r Runner) cmdSearch(args []string) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help") {
		WriteSearchHelp(r.Out)
		return nil
	}
	opts, queryParts, err := parseSearchArgs(args)
	if err != nil {
		return err
	}
	if len(queryParts) == 0 {
		return usageError{"Use: c3po ast search [--path dir] [--glob pat] [--offset N] [--regex] [-i] <query>"}
	}
	query := strings.Join(queryParts, " ")
	if strings.HasPrefix(query, "-") {
		return usageError{"query looks like a flag; see rgw-ast search --help"}
	}
	cfg, _, wsRoot, err := r.workspace()
	if err != nil {
		return err
	}
	res, err := intel.SearchPage(wsRoot, query, cfg, opts)
	if err != nil {
		return err
	}
	for _, h := range res.Hits {
		_, _ = fmt.Fprintf(r.Out, "%s:%d:%s\n", h.Path, h.Line, h.Content)
	}
	if res.More() {
		total := fmt.Sprint(res.Total)
		if res.TotalCapped {
			total += "+"
		}
		_, _ = fmt.Fprintf(r.Err, "truncated: hits %d-%d of %s; next: --offset %d\n",
			res.Offset+1, res.Offset+len(res.Hits), total, res.Offset+len(res.Hits))
	}
	if len(res.Hits) == 0 {
		if res.Total > 0 {
			_, _ = fmt.Fprintf(r.Err, "no hits at --offset %d (%d total)\n", res.Offset, res.Total)
		} else {
			_, _ = fmt.Fprintf(r.Err, "no hits (check include globs, --path, nested-git skips; try rgw-ast explain <path>)\n")
		}
	}
	return nil
}

func parseSearchArgs(args []string) (opts intel.SearchOpts, query []string, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--path" || a == "--glob" || a == "--offset":
			if i+1 >= len(args) {
				return opts, nil, usageError{a + " requires a value"}
			}
			if err := setSearchFlag(&opts, a, args[i+1]); err != nil {
				return opts, nil, err
			}
			i++
		case strings.HasPrefix(a, "--path=") || strings.HasPrefix(a, "--glob=") || strings.HasPrefix(a, "--offset="):
			name, value, _ := strings.Cut(a, "=")
			if err := setSearchFlag(&opts, name, value); err != nil {
				return opts, nil, err
			}
		case a == "--regex" || a == "-E":
			opts.Regex = true
		case a == "-i" || a == "--ignore-case":
			opts.IgnoreCase = true
		case a == "--help" || a == "-h":
			return opts, nil, usageError{"use: rgw-ast search --help"}
		case strings.HasPrefix(a, "-"):
			return opts, nil, usageError{fmt.Sprintf("unknown search flag %q", a)}
		default:
			query = append(query, a)
		}
	}
	return opts, query, nil
}

func setSearchFlag(opts *intel.SearchOpts, name, value string) error {
	switch name {
	case "--path":
		opts.PathPrefix = value
	case "--glob":
		opts.Glob = value
	case "--offset":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return usageError{"--offset requires a non-negative integer"}
		}
		opts.Offset = n
	}
	return nil
}

func (r Runner) cmdExplain(args []string) error {
	if len(args) != 1 {
		return usageError{"Use: c3po ast explain <path>"}
	}
	cfg, _, wsRoot, err := r.workspaceFor(args[0])
	if err != nil {
		return err
	}
	out, err := intel.Explain(wsRoot, args[0], cfg)
	if err != nil {
		return err
	}
	_, _ = io.WriteString(r.Out, out)
	return nil
}

func (r Runner) cmdHash(args []string) error {
	if len(args) < 1 {
		return usageError{"Use: c3po ast hash <file> [<file>...]"}
	}
	for _, p := range args {
		_, _, wsRoot, err := r.workspaceFor(p)
		if err != nil {
			return err
		}
		abs, err := files.ResolvePath(wsRoot, p)
		if err != nil {
			return err
		}
		h, err := files.HashFile(abs)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(wsRoot, abs)
		if err != nil {
			rel = abs
		}
		_, _ = fmt.Fprintf(r.Out, "%s  %s\n", h, filepath.ToSlash(rel))
	}
	return nil
}

func (r Runner) cmdRead(args []string) error {
	path, linesSpec, number, strict, err := parseReadArgs(args)
	if err != nil {
		return err
	}
	cfg, _, wsRoot, err := r.workspaceFor(path)
	if err != nil {
		return err
	}
	m, err := r.measureFresh(cfg, wsRoot)
	if err != nil {
		return err
	}
	d := enforce.Decide(cfg, m.LOC)
	abs, err := files.ResolvePath(wsRoot, path)
	if err != nil {
		return err
	}
	rel, _ := filepath.Rel(wsRoot, abs)
	rel = filepath.ToSlash(rel)

	if linesSpec == "" {
		if d.Enforced && cfg.Enforcement.DenyWholeFileRead {
			bin, err := files.IsBinary(abs)
			if err != nil {
				return err
			}
			if !bin {
				n, _ := files.LineCount(abs)
				return fmt.Errorf("whole-file read denied while enforced (%s has %d lines); use rgw-ast read %s --lines 1-%d, or map/show/search", rel, n, path, min(n, cfg.Enforcement.MaxReadLines))
			}
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		_, _ = r.Out.Write(data)
		if len(data) > 0 && data[len(data)-1] != '\n' {
			_, _ = fmt.Fprintln(r.Out)
		}
		return nil
	}
	start, end, err := parseLineRange(linesSpec)
	if err != nil {
		return usageError{err.Error()}
	}
	text, next, err := files.FormatReadRange(abs, rel, start, end, cfg.Enforcement.MaxReadLines, number, strict)
	if err != nil {
		return err
	}
	_, _ = io.WriteString(r.Out, text)
	if next > 0 {
		_, _ = fmt.Fprintf(r.Err, "next_start: %d\n", next)
	}
	return nil
}

func (r Runner) cmdCreate(args []string) error {
	path, fromFile, stdin, parents, expectAbsent, err := parseCreateArgs(args)
	if err != nil {
		return err
	}
	if !expectAbsent {
		return usageError{"--expect-absent is required"}
	}
	if fromFile == "" && !stdin {
		return usageError{"provide --from-file or --stdin"}
	}
	_, _, wsRoot, err := r.workspaceFor(path)
	if err != nil {
		return err
	}
	abs, err := files.ResolvePath(wsRoot, path)
	if err != nil {
		return err
	}
	var content []byte
	if stdin {
		content, err = files.ReadStdin(r.In)
	} else {
		content, err = files.ReadAllFile(fromFile)
	}
	if err != nil {
		return err
	}
	h, err := files.CreateAbsent(abs, content, parents, 0o644)
	if err != nil {
		return err
	}
	return r.notedHash(wsRoot, abs, h)
}

func (r Runner) cmdAppend(args []string) error {
	path, hash, fromFile, stdin, err := parseAppendArgs(args)
	if err != nil {
		return err
	}
	if hash == "" {
		return usageError{"--expect-hash is required"}
	}
	_, _, wsRoot, err := r.workspaceFor(path)
	if err != nil {
		return err
	}
	abs, err := files.ResolvePath(wsRoot, path)
	if err != nil {
		return err
	}
	var content []byte
	if stdin {
		content, err = files.ReadStdin(r.In)
	} else {
		content, err = files.ReadAllFile(fromFile)
	}
	if err != nil {
		return err
	}
	h, err := files.AppendExact(abs, hash, content)
	if err != nil {
		return err
	}
	rel, _ := filepath.Rel(wsRoot, abs)
	_, _ = fmt.Fprintf(r.Out, "ok  %s  %s\n", h, filepath.ToSlash(rel))
	return nil
}

func (r Runner) cmdPatch(args []string) error {
	symbol, args, err := takeFlagValue(args, "--symbol")
	if err != nil {
		return err
	}
	args, all, err := stripBoolFlag(args, "--all")
	if err != nil {
		return usageError{err.Error()}
	}
	path, expectHash, old, newText, oldFile, newFile, opsFile, err := parsePatchArgs(args)
	if err != nil {
		return err
	}
	if all && symbol != "" {
		return usageError{"--all cannot be combined with --symbol"}
	}
	cfg, _, wsRoot, err := r.workspaceFor(path)
	if err != nil {
		return err
	}
	// recheck enforcement with fresh measure (mutations must not use stale LOC)
	m, err := r.measureFresh(cfg, wsRoot)
	if err != nil {
		return err
	}
	d := enforce.Decide(cfg, m.LOC)
	if d.Enforced && cfg.Enforcement.RequireHashBeforePatch && expectHash == "" {
		return usageError{"--expect-hash is required while enforced"}
	}
	if expectHash == "" {
		return usageError{"--expect-hash is required"}
	}
	abs, err := files.ResolvePath(wsRoot, path)
	if err != nil {
		return err
	}
	var ops []files.Op
	switch {
	case opsFile != "":
		ops, err = files.ParseOpsFile(opsFile)
		if err != nil {
			return err
		}
	case oldFile != "" || newFile != "":
		if oldFile == "" || newFile == "" {
			return usageError{"both --old-file and --new-file are required together"}
		}
		ob, err := files.ReadAllFile(oldFile)
		if err != nil {
			return err
		}
		nb, err := files.ReadAllFile(newFile)
		if err != nil {
			return err
		}
		ops = []files.Op{{Old: string(ob), New: string(nb), All: all}}
	default:
		if old == "" {
			return usageError{"--old or --old-file or --ops-file is required"}
		}
		ops = []files.Op{{Old: old, New: newText, All: all}}
	}
	if symbol != "" {
		results, err := files.ApplyAll(wsRoot, []files.FileChange{{
			Path: path, ExpectHash: expectHash, Old: ops[0].Old, New: ops[0].New, Symbol: symbol,
		}})
		if err != nil {
			return err
		}
		return r.notedSymbol(results)
	}
	newHash, err := files.PatchOps(abs, expectHash, ops)
	if err != nil {
		return err
	}
	return r.notedHash(wsRoot, abs, newHash)
}

// formatterName reports the formatter that ran, or none.
func formatterName(tool string) string {
	if tool == "" {
		return "none"
	}
	return tool
}

func (r Runner) cmdApply(args []string) error {
	dry := false
	from := ""
	var opts files.ApplyOpts
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			dry = true
		case "--no-format":
			opts.NoFormat = true
		case "--from-file":
			if i+1 >= len(args) {
				return usageError{"--from-file requires a path"}
			}
			from = args[i+1]
			i++
		default:
			return usageError{"Use: c3po ast apply --from-file <manifest.json|diff> [--dry-run] [--no-format]"}
		}
	}
	if from == "" {
		return usageError{"Use: c3po ast apply --from-file <manifest.json|diff> [--dry-run] [--no-format]"}
	}
	changes, err := files.ParseApplyFile(from)
	if err != nil {
		return err
	}
	_, _, wsRoot, err := r.workspace()
	if err != nil {
		return err
	}
	if dry {
		results, err := files.DryRunOpts(wsRoot, changes, opts)
		if err != nil {
			return err
		}
		for _, res := range results {
			_, _ = fmt.Fprintf(r.Out, "plan %s %s%s\n", res.Op, res.Path, arrowTo(res.To))
			_, _ = fmt.Fprintf(r.Out, "formatter %s\n", formatterName(res.Formatter))
			if res.Diff != "" {
				_, _ = fmt.Fprintln(r.Out, res.Diff)
			}
		}
		return nil
	}
	results, err := files.ApplyAllOpts(wsRoot, changes, opts)
	if err != nil {
		return err
	}
	for _, res := range results {
		_, _ = fmt.Fprintf(r.Out, "ok  %s  %s%s", res.Hash, res.Path, arrowTo(res.To))
		if res.Op != "" && res.Op != "edit" {
			_, _ = fmt.Fprintf(r.Out, "  %s", res.Op)
		}
		if res.Op != "delete" && res.Op != "move" {
			_, _ = fmt.Fprintf(r.Out, "  formatter %s", formatterName(res.Formatter))
		}
		_, _ = fmt.Fprintln(r.Out)
	}
	return nil
}

func arrowTo(to string) string {
	if to == "" {
		return ""
	}
	return " -> " + to
}

func takeFlagValue(args []string, name string) (string, []string, error) {
	val := ""
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == name {
			if i+1 >= len(args) {
				return "", nil, usageError{name + " requires a value"}
			}
			if val != "" {
				return "", nil, usageError{"duplicate " + name}
			}
			val = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(a, name+"=") {
			if val != "" {
				return "", nil, usageError{"duplicate " + name}
			}
			val = strings.TrimPrefix(a, name+"=")
			continue
		}
		rest = append(rest, a)
	}
	return val, rest, nil
}

func (r Runner) cmdDelete(args []string) error {
	path, expectHash, pruneEmpty, err := parseDeleteArgs(args)
	if err != nil {
		return err
	}
	if expectHash == "" {
		return usageError{"--expect-hash is required"}
	}
	cfg, _, wsRoot, err := r.workspaceFor(path)
	if err != nil {
		return err
	}
	if _, err := r.measureFresh(cfg, wsRoot); err != nil {
		return err
	}
	abs, err := files.ResolvePath(wsRoot, path)
	if err != nil {
		return err
	}
	pruned, err := files.DeleteExact(wsRoot, abs, expectHash, pruneEmpty)
	if err != nil {
		return err
	}
	rel, _ := filepath.Rel(wsRoot, abs)
	_, _ = fmt.Fprintf(r.Out, "ok  deleted  %s\n", filepath.ToSlash(rel))
	for _, dir := range pruned {
		rel, _ := filepath.Rel(wsRoot, dir)
		_, _ = fmt.Fprintf(r.Out, "ok  pruned  %s\n", filepath.ToSlash(rel))
	}
	return nil
}

func (r Runner) cmdMove(args []string) error {
	args, parents, err := stripBoolFlag(args, "--parents")
	if err != nil {
		return usageError{err.Error()}
	}
	expectHash, args, err := takeFlagValue(args, "--expect-hash")
	if err != nil {
		return err
	}
	if len(args) != 2 {
		return usageError{"Use: c3po ast move <src> <dst> --expect-hash <sha> [--parents]"}
	}
	if expectHash == "" {
		return usageError{"--expect-hash is required"}
	}
	cfg, _, wsRoot, err := r.workspaceFor(args[0])
	if err != nil {
		return err
	}
	if _, err := r.measureFresh(cfg, wsRoot); err != nil {
		return err
	}
	h, viaGit, err := files.MoveExact(wsRoot, args[0], args[1], expectHash, parents)
	if err != nil {
		return err
	}
	rel := func(p string) string {
		abs, err := files.ResolvePath(wsRoot, p)
		if err != nil {
			return p
		}
		out, _ := filepath.Rel(wsRoot, abs)
		return filepath.ToSlash(out)
	}
	how := "renamed"
	if viaGit {
		how = "git mv"
	}
	_, _ = fmt.Fprintf(r.Out, "ok  %s  %s -> %s  %s\n", h, rel(args[0]), rel(args[1]), how)
	return nil
}

func (r Runner) cmdHook(args []string) error {
	if len(args) != 0 {
		return usageError{"Use: c3po ast hook"}
	}
	cfg, _, err := r.load()
	if err != nil {
		_ = json.NewEncoder(r.Out).Encode(map[string]any{
			"permissionDecision": "deny", "permissionDecisionReason": err.Error(),
		})
		return nil
	}
	cwd := r.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	in := r.In
	if in == nil {
		in = os.Stdin
	}
	if err := hook.Run(in, r.Out, cfg, cwd, r.RootOverride); err != nil {
		_ = json.NewEncoder(r.Out).Encode(map[string]any{
			"permissionDecision": "deny", "permissionDecisionReason": err.Error(),
		})
	}
	return nil
}

func (r Runner) cmdExec(args []string) error {
	// c3po ast exec [--json] -- <command...>
	asJSON := false
	var cmdArgs []string
	seenSep := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			seenSep = true
			cmdArgs = args[i+1:]
			break
		}
		if a == "--json" {
			asJSON = true
			continue
		}
		return usageError{"Use: c3po ast exec [--json] -- <command...>"}
	}
	if !seenSep || len(cmdArgs) == 0 {
		return usageError{"Use: c3po ast exec [--json] -- <command...>"}
	}
	cfg, _, wsRoot, err := r.workspace()
	if err != nil {
		return err
	}
	m, err := r.measureFresh(cfg, wsRoot)
	if err != nil {
		return err
	}
	d := enforce.Decide(cfg, m.LOC)
	rep, err := execgen.Run(wsRoot, cfg, cmdArgs, d.Enforced)
	if err != nil {
		return err
	}
	if asJSON {
		if err := writeJSON(r.Out, rep); err != nil {
			return err
		}
	} else {
		_, _ = fmt.Fprintf(r.Out, "exit_code: %d\nallowed_by: %s\nchanges: %d\n", rep.ExitCode, rep.AllowedBy, len(rep.Changes))
		for _, c := range rep.Changes {
			if c.Hash != "" {
				_, _ = fmt.Fprintf(r.Out, "  %s  %s  %s\n", c.Status, c.Hash, c.Path)
			} else {
				_, _ = fmt.Fprintf(r.Out, "  %s  %s\n", c.Status, c.Path)
			}
		}
	}
	if rep.ExitCode != 0 {
		return fmt.Errorf("generator exit %d", rep.ExitCode)
	}
	return nil
}

func (r Runner) cmdDoctor(args []string) error {
	asJSON, rest, err := takeJSONFlag(args)
	if err != nil {
		return err
	}
	rest, refresh, err := stripBoolFlag(rest, "--refresh")
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return usageError{"Use: c3po ast doctor [--json] [--refresh]"}
	}
	cfg, cfgPath, err := r.load()
	if err != nil {
		return err
	}
	cwd := r.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	rep := doctor.Run(cfg, cfgPath, cwd, r.RootOverride, refresh || r.Refresh)
	if asJSON {
		b, err := doctor.FormatJSON(rep)
		if err != nil {
			return err
		}
		_, _ = r.Out.Write(b)
		_, _ = fmt.Fprintln(r.Out)
	} else {
		_, _ = io.WriteString(r.Out, doctor.FormatText(rep))
	}
	if !rep.OK {
		if rep.EnforcementStatus == "degraded" && len(rep.Errors) == 0 {
			return fmt.Errorf("enforcement degraded")
		}
		return fmt.Errorf("doctor reported errors")
	}
	return nil
}

func (r Runner) cmdAgentsBlock(args []string) error {
	if len(args) != 0 {
		return usageError{"Use: c3po ast agents-block"}
	}
	_, _ = io.WriteString(r.Out, agents.Block)
	if !strings.HasSuffix(agents.Block, "\n") {
		_, _ = fmt.Fprintln(r.Out)
	}
	return nil
}

func (r Runner) cmdPackages(args []string) error {
	asJSON, rest, err := takeJSONFlag(args)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return usageError{"Use: c3po ast packages [--json]"}
	}
	_, _, wsRoot, err := r.workspace()
	if err != nil {
		return err
	}
	pkgs, err := packages.Discover(wsRoot)
	if err != nil {
		return err
	}
	if asJSON {
		return writeJSON(r.Out, pkgs)
	}
	printPackages(r.Out, pkgs)
	return nil
}

func (r Runner) cmdAffected(args []string) error {
	asJSON, rest, err := takeJSONFlag(args)
	if err != nil {
		return err
	}
	_, _, wsRoot, err := r.workspace()
	if err != nil {
		return err
	}
	pkgs, err := packages.Discover(wsRoot)
	if err != nil {
		return err
	}
	paths := rest
	if len(paths) == 0 {
		paths, err = packages.GitChangedPaths(wsRoot)
		if err != nil {
			return err
		}
	}
	got := packages.MapPaths(pkgs, paths)
	if g, err := graph.Build(wsRoot, pkgs); err == nil {
		extra := g.ReverseImportReach(paths)
		got = graph.MergeAffected(got, extra, pkgs)
	}
	if asJSON {
		return writeJSON(r.Out, got)
	}
	if len(got.Packages) == 0 {
		_, _ = fmt.Fprintln(r.Out, "nothing affected")
		return nil
	}
	printPackages(r.Out, got.Packages)
	if len(got.Unmapped) > 0 {
		_, _ = fmt.Fprintln(r.Out, "unmapped:")
		for _, p := range got.Unmapped {
			_, _ = fmt.Fprintln(r.Out, "  "+p)
		}
	}
	return nil
}

func printPackages(w io.Writer, pkgs []packages.Package) {
	for _, p := range pkgs {
		lang := strings.Join(p.Language, ",")
		if lang == "" {
			lang = "-"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", p.Path, p.Kind, lang)
		for _, v := range p.Verify {
			_, _ = fmt.Fprintf(w, "  verify: %s\n", v)
		}
		if p.Install != "" {
			_, _ = fmt.Fprintf(w, "  install: %s\n", p.Install)
		}
	}
}

func (r Runner) loadGraph() (*graph.Graph, []packages.Package, error) {
	_, _, wsRoot, err := r.workspace()
	if err != nil {
		return nil, nil, err
	}
	pkgs, err := packages.Discover(wsRoot)
	if err != nil {
		return nil, nil, err
	}
	g, err := graph.Build(wsRoot, pkgs)
	if err != nil {
		return nil, nil, err
	}
	return g, pkgs, nil
}

func (r Runner) cmdGraph(args []string) error {
	asJSON, rest, err := takeJSONFlag(args)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return usageError{"Use: c3po ast graph [--json]"}
	}
	g, _, err := r.loadGraph()
	if err != nil {
		return err
	}
	kinds, rels, nNodes, nEdges := g.Counts()
	nodes, edges := g.Snapshot()
	comps := g.ImportComponents()
	top := g.TopGopkgs(8)
	type topN struct {
		Path   string `json:"path"`
		Degree int    `json:"degree"`
	}
	var tops []topN
	for _, n := range top {
		tops = append(tops, topN{Path: n.Path, Degree: g.Degree(n.ID)})
	}
	if nNodes != len(nodes) || nEdges != len(edges) {
		return fmt.Errorf("graph counts do not match the node and edge arrays")
	}
	out := map[string]any{
		"nodes":           nodes,
		"edges":           edges,
		"node_count":      nNodes,
		"edge_count":      nEdges,
		"kinds":           kinds,
		"relations":       rels,
		"components":      comps,
		"component_count": len(comps),
		"top":             tops,
	}
	if asJSON {
		return writeJSON(r.Out, out)
	}
	_, _ = fmt.Fprintf(r.Out, "nodes: %d\nedges: %d\ncomponents: %d\n", nNodes, nEdges, len(comps))
	_, _ = fmt.Fprintf(r.Out, "json lists the node and edge arrays; these counts must match them\n")
	for _, k := range sortedKeys(kinds) {
		_, _ = fmt.Fprintf(r.Out, "  kind %s: %d\n", k, kinds[k])
	}
	for _, k := range sortedKeys(rels) {
		_, _ = fmt.Fprintf(r.Out, "  rel %s: %d\n", k, rels[k])
	}
	for _, t := range tops {
		_, _ = fmt.Fprintf(r.Out, "  top %s degree %d\n", t.Path, t.Degree)
	}
	return nil
}

func (r Runner) cmdNeighbors(args []string) error {
	asJSON, rest, err := takeJSONFlag(args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageError{"Use: c3po ast neighbors [--json] <id>"}
	}
	g, _, err := r.loadGraph()
	if err != nil {
		return err
	}
	n, ok := g.Resolve(rest[0])
	if !ok {
		return fmt.Errorf("unknown node %q", rest[0])
	}
	nb := g.Neighbors(n.ID)
	ring := g.RingHash(n.ID)
	if asJSON {
		return writeJSON(r.Out, map[string]any{
			"id": n.ID, "kind": n.Kind, "path": n.Path,
			"neighbors": nb, "ring_hash": ring, "degree": g.Degree(n.ID),
		})
	}
	_, _ = fmt.Fprintf(r.Out, "%s\tring_hash %s\tdegree %d\n", n.ID, ring, g.Degree(n.ID))
	for _, e := range nb {
		_, _ = fmt.Fprintf(r.Out, "  %s %s -> %s\n", e.Rel, e.From, e.To)
	}
	return nil
}

func (r Runner) cmdCut(args []string) error {
	asJSON, rest, err := takeJSONFlag(args)
	if err != nil {
		return err
	}
	if len(rest) > 1 {
		return usageError{"Use: c3po ast cut [--json] [path]"}
	}
	g, _, err := r.loadGraph()
	if err != nil {
		return err
	}
	cuts := g.CutVertices()
	if len(rest) == 1 {
		n, ok := g.Resolve(rest[0])
		id := rest[0]
		if ok {
			if n.Kind == "file" {
				id = graph.ID("gopkg", strings.TrimPrefix(filepath.ToSlash(filepath.Dir(n.Path)), "./"))
			} else if n.Kind == "gopkg" {
				id = n.ID
			} else {
				id = graph.ID("gopkg", n.Path)
			}
		}
		isCut := false
		for _, c := range cuts {
			if c == id {
				isCut = true
				break
			}
		}
		if asJSON {
			return writeJSON(r.Out, map[string]any{"id": id, "cut": isCut, "cuts": cuts})
		}
		if isCut {
			_, _ = fmt.Fprintf(r.Out, "cut: %s\n", id)
		} else {
			_, _ = fmt.Fprintf(r.Out, "not-cut: %s\n", id)
		}
		return nil
	}
	if asJSON {
		return writeJSON(r.Out, map[string]any{"cuts": cuts})
	}
	if len(cuts) == 0 {
		_, _ = fmt.Fprintln(r.Out, "no cut vertices")
		return nil
	}
	for _, c := range cuts {
		_, _ = fmt.Fprintln(r.Out, c)
	}
	return nil
}

func (r Runner) cmdTrace(mode string, args []string) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		WriteTraceHelp(r.Out)
		return nil
	}
	asJSON, rest, err := takeJSONFlag(args)
	if err != nil {
		return err
	}
	query, limit, offset, err := parseTraceArgs(rest)
	if err != nil {
		return err
	}
	_, _, wsRoot, err := r.workspace()
	if err != nil {
		return err
	}
	cfg, _, err := r.load()
	if err != nil {
		return err
	}
	res, err := trace.Query(wsRoot, cfg, query, mode, limit, offset)
	if err != nil {
		return err
	}
	if res.Ambiguous {
		if asJSON {
			if werr := writeJSON(r.Out, res); werr != nil {
				return werr
			}
		} else {
			_, _ = io.WriteString(r.Out, trace.FormatText(res))
		}
		return fmt.Errorf("ambiguous symbol %q", query)
	}
	if asJSON {
		return writeJSON(r.Out, res)
	}
	_, _ = io.WriteString(r.Out, trace.FormatText(res))
	return nil
}

func parseTraceArgs(args []string) (query string, limit, offset int, err error) {
	limit = 0
	offset = 0
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--limit", "--offset":
			if i+1 >= len(args) {
				return "", 0, 0, usageError{a + " requires a number"}
			}
			n, nerr := strconv.Atoi(args[i+1])
			if nerr != nil || n < 0 {
				return "", 0, 0, usageError{a + " requires a non-negative number"}
			}
			if a == "--limit" {
				limit = n
			} else {
				offset = n
			}
			i++
		default:
			if strings.HasPrefix(a, "-") {
				return "", 0, 0, usageError{"unknown flag " + a}
			}
			pos = append(pos, a)
		}
	}
	if len(pos) != 1 {
		return "", 0, 0, usageError{"Use: c3po ast callers|callees|impact [--json] [--limit N] [--offset N] <symbol>"}
	}
	return pos[0], limit, offset, nil
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func takeJSONFlag(args []string) (bool, []string, error) {
	asJSON := false
	var rest []string
	for _, a := range args {
		if a == "--json" {
			if asJSON {
				return false, nil, usageError{"duplicate --json"}
			}
			asJSON = true
			continue
		}
		rest = append(rest, a)
	}
	return asJSON, rest, nil
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func parseReadArgs(args []string) (path, lines string, number, strict bool, err error) {
	if len(args) == 0 {
		return "", "", false, false, usageError{"Use: c3po ast read <file> --lines <start>-<end>"}
	}
	path = args[0]
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--lines":
			if i+1 >= len(args) {
				return "", "", false, false, usageError{"--lines requires START-END"}
			}
			lines = args[i+1]
			i++
		case strings.HasPrefix(a, "--lines="):
			lines = strings.TrimPrefix(a, "--lines=")
		case a == "--number" || a == "-n":
			number = true
		case a == "--strict-lines":
			strict = true
		default:
			return "", "", false, false, usageError{fmt.Sprintf("unexpected argument %q", a)}
		}
	}
	return path, lines, number, strict, nil
}

func parseLineRange(s string) (start, end int, err error) {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("lines must be START-END")
	}
	start, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start line")
	}
	end, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end line")
	}
	if start < 1 || end < start {
		return 0, 0, fmt.Errorf("invalid line range %s", s)
	}
	return start, end, nil
}

func parseCreateArgs(args []string) (path, fromFile string, stdin, parents, expectAbsent bool, err error) {
	if len(args) < 1 {
		return "", "", false, false, false, usageError{"Use: c3po ast create <file> --expect-absent ..."}
	}
	path = args[0]
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--expect-absent":
			expectAbsent = true
		case a == "--parents":
			parents = true
		case a == "--stdin":
			stdin = true
		case a == "--from-file":
			if i+1 >= len(args) {
				return "", "", false, false, false, usageError{"--from-file requires a path"}
			}
			fromFile = args[i+1]
			i++
		case strings.HasPrefix(a, "--from-file="):
			fromFile = strings.TrimPrefix(a, "--from-file=")
		default:
			return "", "", false, false, false, usageError{fmt.Sprintf("unexpected %q", a)}
		}
	}
	return path, fromFile, stdin, parents, expectAbsent, nil
}

func parseAppendArgs(args []string) (path, hash, fromFile string, stdin bool, err error) {
	if len(args) < 1 {
		return "", "", "", false, usageError{"Use: c3po ast append <file> --expect-hash <sha> ..."}
	}
	path = args[0]
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--expect-hash":
			if i+1 >= len(args) {
				return "", "", "", false, usageError{"--expect-hash requires a value"}
			}
			hash = args[i+1]
			i++
		case strings.HasPrefix(a, "--expect-hash="):
			hash = strings.TrimPrefix(a, "--expect-hash=")
		case a == "--stdin":
			stdin = true
		case a == "--from-file":
			if i+1 >= len(args) {
				return "", "", "", false, usageError{"--from-file requires a path"}
			}
			fromFile = args[i+1]
			i++
		case strings.HasPrefix(a, "--from-file="):
			fromFile = strings.TrimPrefix(a, "--from-file=")
		default:
			return "", "", "", false, usageError{fmt.Sprintf("unexpected %q", a)}
		}
	}
	if fromFile == "" && !stdin {
		return "", "", "", false, usageError{"provide --from-file or --stdin"}
	}
	return path, hash, fromFile, stdin, nil
}

func parseDeleteArgs(args []string) (path, hash string, pruneEmpty bool, err error) {
	if len(args) < 1 {
		return "", "", false, usageError{"Use: c3po ast delete <file> --expect-hash <sha> [--prune-empty]"}
	}
	path = args[0]
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--expect-hash":
			if i+1 >= len(args) {
				return "", "", false, usageError{"--expect-hash requires a value"}
			}
			hash = args[i+1]
			i++
		case strings.HasPrefix(a, "--expect-hash="):
			hash = strings.TrimPrefix(a, "--expect-hash=")
		case a == "--prune-empty":
			pruneEmpty = true
		default:
			return "", "", false, usageError{fmt.Sprintf("unexpected argument %q", a)}
		}
	}
	return path, hash, pruneEmpty, nil
}

func parsePatchArgs(args []string) (path, hash, old, new, oldFile, newFile, opsFile string, err error) {
	if len(args) < 1 {
		return "", "", "", "", "", "", "", usageError{"Use: c3po ast patch <file> --expect-hash <sha> ..."}
	}
	path = args[0]
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--expect-hash":
			if i+1 >= len(args) {
				return "", "", "", "", "", "", "", usageError{"--expect-hash requires a value"}
			}
			hash = args[i+1]
			i++
		case strings.HasPrefix(a, "--expect-hash="):
			hash = strings.TrimPrefix(a, "--expect-hash=")
		case a == "--old":
			if i+1 >= len(args) {
				return "", "", "", "", "", "", "", usageError{"--old requires a value"}
			}
			old = args[i+1]
			i++
		case strings.HasPrefix(a, "--old="):
			old = strings.TrimPrefix(a, "--old=")
		case a == "--new":
			if i+1 >= len(args) {
				return "", "", "", "", "", "", "", usageError{"--new requires a value"}
			}
			new = args[i+1]
			i++
		case strings.HasPrefix(a, "--new="):
			new = strings.TrimPrefix(a, "--new=")
		case a == "--old-file":
			if i+1 >= len(args) {
				return "", "", "", "", "", "", "", usageError{"--old-file requires a path"}
			}
			oldFile = args[i+1]
			i++
		case strings.HasPrefix(a, "--old-file="):
			oldFile = strings.TrimPrefix(a, "--old-file=")
		case a == "--new-file":
			if i+1 >= len(args) {
				return "", "", "", "", "", "", "", usageError{"--new-file requires a path"}
			}
			newFile = args[i+1]
			i++
		case strings.HasPrefix(a, "--new-file="):
			newFile = strings.TrimPrefix(a, "--new-file=")
		case a == "--ops-file":
			if i+1 >= len(args) {
				return "", "", "", "", "", "", "", usageError{"--ops-file requires a path"}
			}
			opsFile = args[i+1]
			i++
		case strings.HasPrefix(a, "--ops-file="):
			opsFile = strings.TrimPrefix(a, "--ops-file=")
		default:
			return "", "", "", "", "", "", "", usageError{fmt.Sprintf("unexpected argument %q", a)}
		}
	}
	return path, hash, old, new, oldFile, newFile, opsFile, nil
}
