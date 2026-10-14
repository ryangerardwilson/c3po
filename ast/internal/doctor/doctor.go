package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/ast/internal/agents"
	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/enforce"
	"github.com/ryangerardwilson/c3po/ast/internal/measure"
	"github.com/ryangerardwilson/c3po/ast/internal/root"
	"github.com/ryangerardwilson/c3po/ast/internal/version"
)

// Report is a bootstrap/readiness inspection for a workspace root.
type Report struct {
	OK                bool     `json:"ok"`
	Ready             bool     `json:"ready"`
	EnforcementStatus string   `json:"enforcement_status"`
	Version           string   `json:"version"`
	Commit            string   `json:"commit"`
	BuildTime         string   `json:"build_time"`
	ConfigPath        string   `json:"config_path"`
	ConfigExists      bool     `json:"config_exists"`
	Root              string   `json:"root"`
	LOC               int      `json:"loc"`
	Enforced          bool     `json:"enforced"`
	Mode              string   `json:"mode"`
	ThresholdLOC      int      `json:"threshold_loc"`
	CacheHit          bool     `json:"cache_hit"`
	GitIgnoreActive   bool     `json:"gitignore_active"`
	NestedRepos       int      `json:"nested_repos_skipped"`
	GitAvailable      bool     `json:"git_available"`
	HookReady         bool     `json:"hook_ready"`
	HookStatus        string   `json:"hook_status"` // configured|not_checked
	AgentsBlockReady  bool     `json:"agents_block_ready"`
	AgentsBlockStatus string   `json:"agents_block_status"` // current|inherited|stale|missing_block|malformed|absent_file
	GeneratorsAllow   int      `json:"generators_allow_count"`
	Notes             []string `json:"notes,omitempty"`
	Errors            []string `json:"errors,omitempty"`
}

// Run gathers diagnostics for cwd/rootOverride.
func Run(cfg config.Config, cfgPath, cwd, rootOverride string, refresh bool) Report {
	rep := Report{
		Version:           version.Version,
		Commit:            version.Commit,
		BuildTime:         version.BuildTime,
		ConfigPath:        cfgPath,
		HookReady:         false,
		HookStatus:        "not_checked",
		AgentsBlockReady:  false,
		AgentsBlockStatus: "absent_file",
		GeneratorsAllow:   len(cfg.Generators.Allow),
	}
	if _, err := os.Stat(cfgPath); err == nil {
		rep.ConfigExists = true
	} else {
		rep.Errors = append(rep.Errors, "config missing: "+cfgPath)
	}

	ws := rootOverride
	var err error
	if ws == "" {
		ws, err = root.Resolve(cwd)
		if err != nil {
			rep.Errors = append(rep.Errors, err.Error())
			rep.OK = false
			return rep
		}
	} else {
		ws, err = filepath.Abs(ws)
		if err != nil {
			rep.Errors = append(rep.Errors, err.Error())
			rep.OK = false
			return rep
		}
	}
	rep.Root = ws

	m, err := measure.CountCachedOpts(ws, cfg, measure.CountOpts{Refresh: refresh})
	if err != nil {
		rep.Errors = append(rep.Errors, "measure: "+err.Error())
	} else {
		d := enforce.Decide(cfg, m.LOC)
		rep.LOC = m.LOC
		rep.Enforced = d.Enforced
		rep.Mode = d.Mode
		rep.ThresholdLOC = d.ThresholdLOC
		rep.CacheHit = m.CacheHit
		rep.GitIgnoreActive = m.GitIgnoreActive
		rep.NestedRepos = m.NestedReposSkipped
	}

	if _, err := exec.LookPath("git"); err == nil {
		rep.GitAvailable = true
	} else {
		rep.Notes = append(rep.Notes, "git not on PATH; dirty-tree fingerprint uses mtime sampling")
	}

	// AGENTS.md managed block check against the canonical emitted block.
	agentsPath := filepath.Join(ws, "AGENTS.md")
	if data, err := os.ReadFile(agentsPath); err != nil {
		rep.AgentsBlockStatus = "absent_file"
		rep.AgentsBlockReady = false
		if rep.Enforced {
			rep.Notes = append(rep.Notes, "AGENTS.md missing; embed with: rgw-ast agents-block")
		}
	} else {
		rep.AgentsBlockStatus, rep.AgentsBlockReady = agentsBlockState(string(data))
		if rep.AgentsBlockStatus == "stale" {
			rep.Notes = append(rep.Notes, "AGENTS.md rgw-ast block differs from canonical output; refresh from: rgw-ast agents-block")
		}
	}
	if rep.AgentsBlockStatus == "absent_file" || rep.AgentsBlockStatus == "missing_block" {
		if from, status := ancestorBlock(ws); status == "current" {
			rep.AgentsBlockStatus, rep.AgentsBlockReady = "inherited", true
			rep.Notes = append(rep.Notes, "agents block inherited from "+from)
		} else if status == "stale" {
			rep.Notes = append(rep.Notes, "ancestor "+from+" has a stale rgw-ast block; refresh from: rgw-ast agents-block")
		}
	}

	// Host hooks: claim configured only for a parsed command field whose argv
	// begins with rgw-ast hook. Mere prose or unrelated JSON strings are not evidence.
	hookFound := false
	for _, p := range hookSettingsPaths(ws) {
		if data, err := os.ReadFile(p); err == nil && hasConfiguredHook(data) {
			hookFound = true
			rep.Notes = append(rep.Notes, "found rgw-ast hook command in "+p)
		}
	}
	if hookFound {
		rep.HookStatus = "configured"
		rep.HookReady = true
	} else {
		rep.HookStatus = "not_checked"
		rep.HookReady = false
		rep.Notes = append(rep.Notes, "host PreToolUse hook not observed; wire: rgw-ast hook")
	}

	if len(cfg.Generators.Allow) == 0 {
		rep.Notes = append(rep.Notes, "generators.allow empty; exec will reject all generators")
	}

	diagnostics := len(rep.Errors) == 0
	switch {
	case !rep.Enforced:
		rep.EnforcementStatus = "inactive"
		rep.Ready = diagnostics
	case rep.HookReady && rep.AgentsBlockReady:
		rep.EnforcementStatus = "ready"
		rep.Ready = diagnostics
	default:
		rep.EnforcementStatus = "degraded"
		rep.Ready = false
		rep.Notes = append(rep.Notes, "enforcement is degraded: the hook and the current agents block are both required while enforced")
	}
	// A degraded enforced checkout is not a healthy result.
	rep.OK = diagnostics && rep.EnforcementStatus != "degraded"
	return rep
}

// ancestorBlock finds the nearest AGENTS.md above ws that has a managed
// block and returns its path and block status. Hosts load ancestor
// AGENTS.md files, so a nested repository can rely on that block.
func ancestorBlock(ws string) (string, string) {
	dir := filepath.Dir(ws)
	for {
		path := filepath.Join(dir, "AGENTS.md")
		if data, err := os.ReadFile(path); err == nil {
			if status, _ := agentsBlockState(string(data)); status != "missing_block" {
				return path, status
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ""
		}
		dir = parent
	}
}

func agentsBlockState(text string) (string, bool) {
	const begin = "<!-- rgw-ast:begin -->"
	const end = "<!-- rgw-ast:end -->"
	if strings.Count(text, begin) == 0 && strings.Count(text, end) == 0 {
		return "missing_block", false
	}
	if strings.Count(text, begin) != 1 || strings.Count(text, end) != 1 {
		return "malformed", false
	}
	start := strings.Index(text, begin)
	endStart := strings.Index(text, end)
	if start < 0 || endStart < start {
		return "malformed", false
	}
	candidate := text[start : endStart+len(end)]
	if normalizeManagedBlock(candidate) == normalizeManagedBlock(agents.Block) {
		return "current", true
	}
	return "stale", false
}

func normalizeManagedBlock(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
}

// hookSettingsPaths lists the workspace host settings, then the user-level
// ones a globally installed hook lives in.
func hookSettingsPaths(ws string) []string {
	paths := []string{
		filepath.Join(ws, ".codex", "hooks.json"),
		filepath.Join(ws, ".claude", "settings.json"),
		filepath.Join(ws, ".claude", "settings.local.json"),
	}
	home, _ := os.UserHomeDir()
	claude := os.Getenv("CLAUDE_CONFIG_DIR")
	if claude == "" && home != "" {
		claude = filepath.Join(home, ".claude")
	}
	codex := os.Getenv("CODEX_HOME")
	if codex == "" && home != "" {
		codex = filepath.Join(home, ".codex")
	}
	if claude != "" {
		paths = append(paths, filepath.Join(claude, "settings.json"))
	}
	if codex != "" {
		paths = append(paths, filepath.Join(codex, "hooks.json"))
	}
	seen := map[string]bool{}
	unique := paths[:0]
	for _, p := range paths {
		if !seen[filepath.Clean(p)] {
			seen[filepath.Clean(p)] = true
			unique = append(unique, p)
		}
	}
	return unique
}

func hasConfiguredHook(data []byte) bool {
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return false
	}
	return containsHookCommand(document)
}

func containsHookCommand(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			switch strings.ToLower(key) {
			case "command", "cmd", "script":
				if isHookCommandValue(child) {
					return true
				}
			}
			if containsHookCommand(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsHookCommand(child) {
				return true
			}
		}
	}
	return false
}

func isHookCommandValue(value any) bool {
	switch typed := value.(type) {
	case string:
		fields := strings.Fields(typed)
		if len(fields) < 2 {
			return false
		}
		first := strings.Trim(fields[0], "\"'")
		second := strings.Trim(fields[1], "\"'")
		return filepath.Base(first) == "rgw-ast" && second == "hook"
	case []any:
		if len(typed) < 2 {
			return false
		}
		first, firstOK := typed[0].(string)
		second, secondOK := typed[1].(string)
		return firstOK && secondOK && filepath.Base(first) == "rgw-ast" && second == "hook"
	default:
		return false
	}
}

// FormatText renders a compact human report.
func FormatText(r Report) string {
	var b strings.Builder
	if r.EnforcementStatus == "degraded" {
		fmt.Fprintf(&b, "degraded: true\n")
	}
	fmt.Fprintf(&b, "enforcement_status: %s\n", r.EnforcementStatus)
	fmt.Fprintf(&b, "ready: %v\n", r.Ready)
	fmt.Fprintf(&b, "ok: %v\n", r.OK)
	fmt.Fprintf(&b, "version: %s\n", version.String())
	fmt.Fprintf(&b, "config: %s (exists=%v)\n", r.ConfigPath, r.ConfigExists)
	fmt.Fprintf(&b, "root: %s\n", r.Root)
	fmt.Fprintf(&b, "loc: %d\nenforced: %v\nmode: %s\nthreshold_loc: %d\n", r.LOC, r.Enforced, r.Mode, r.ThresholdLOC)
	fmt.Fprintf(&b, "cache_hit: %v\ngitignore_active: %v\nnested_repos_skipped: %d\n", r.CacheHit, r.GitIgnoreActive, r.NestedRepos)
	fmt.Fprintf(&b, "git_available: %v\n", r.GitAvailable)
	fmt.Fprintf(&b, "hook_ready: %v\nhook_status: %s\n", r.HookReady, r.HookStatus)
	fmt.Fprintf(&b, "agents_block_ready: %v\nagents_block_status: %s\n", r.AgentsBlockReady, r.AgentsBlockStatus)
	fmt.Fprintf(&b, "generators_allow_count: %d\n", r.GeneratorsAllow)
	for _, n := range r.Notes {
		fmt.Fprintf(&b, "note: %s\n", n)
	}
	for _, e := range r.Errors {
		fmt.Fprintf(&b, "error: %s\n", e)
	}
	return b.String()
}

// FormatJSON encodes the report.
func FormatJSON(r Report) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
