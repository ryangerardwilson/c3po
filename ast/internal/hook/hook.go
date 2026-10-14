package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/enforce"
	"github.com/ryangerardwilson/c3po/ast/internal/measure"
)

const shellDenyDefault = "use rgw-ast patch/create/move/delete, or rgw-ast exec -- <generator>"

// Request is a host pre-tool-use event (Claude/Codex-like).
type Request struct {
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
	Command   string         `json:"command"`
	Name      string         `json:"name"`
	// Cwd is the host session directory (Claude Code sends it on every event).
	Cwd string `json:"cwd"`
}

// Decision is returned to the host.
type Decision struct {
	PermissionDecision       string        `json:"permissionDecision"`
	PermissionDecisionReason string        `json:"permissionDecisionReason,omitempty"`
	HookSpecificOutput       *hookSpecific `json:"hookSpecificOutput,omitempty"`
}

type hookSpecific struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
}

var denyTools = map[string]bool{
	"write": true, "edit": true, "create": true, "delete": true,
	"apply_patch": true, "applypatch": true, "str_replace": true,
	"search_replace": true, "searchreplace": true,
	"notebookedit": true, "notebook_edit": true,
	"createorupdatefile": true, "delete_file": true, "deletefile": true,
	"edit_file": true, "editfile": true, "write_file": true, "writefile": true,
	"multiedit": true, "multi_edit": true,
}

// Evaluate decides allow/deny for a tool call by the enforcement of what it
// would write.
func Evaluate(req Request, cfg config.Config, cwd, rootOverride string) (Decision, error) {
	if rootOverride == "" && strings.TrimSpace(req.Cwd) != "" {
		cwd = req.Cwd
	}
	name := toolName(req)
	tool := normalizeTool(name)
	if tool == "rgwast" {
		return allow(), nil
	}
	j := &judge{cfg: cfg, override: rootOverride, measured: map[string]bool{}}

	if readTools[tool] {
		return j.read(req.ToolInput, cwd)
	}

	if denyTools[tool] {
		target := targetPath(req.ToolInput, cwd)
		if target == "" {
			target = absPath(".", cwd) // no path given: judge the session directory
		}
		repo, enforced, err := j.enforced(target)
		if err != nil || !enforced {
			return allow(), err
		}
		if createTools[tool] {
			if _, err := os.Lstat(target); os.IsNotExist(err) {
				return allow(), nil // a create cannot clobber anything
			}
		}
		return deny(j.fileDenyReason(repo, target, name, cwd)), nil
	}

	if isShellish(name) {
		cmd := strings.TrimSpace(req.Command)
		if cmd == "" && req.ToolInput != nil {
			cmd = firstString(req.ToolInput, "command", "cmd", "script", "code")
		}
		an := analyzeShell(cmd, cwd, &j.cfg)
		for _, g := range an.generators {
			repo, enforced, err := j.enforced(g.cwd)
			if err != nil {
				return Decision{}, err
			}
			if enforced {
				return deny("enforced workspace " + repo + ": generators run through rgw-ast exec so their writes are recorded. Run: rgw-ast exec -- " + g.cmd), nil
			}
		}
		for _, t := range an.targets {
			repo, enforced, err := j.enforced(t.path)
			if err != nil {
				return Decision{}, err
			}
			if enforced {
				hint := t.hint
				if hint == "" {
					hint = shellDenyDefault
				}
				return deny("enforced workspace " + repo + ": shell write to " + t.path + " denied; " + hint), nil
			}
		}
	}
	return allow(), nil
}

// Run reads JSON request from in and writes decision JSON to out.
func Run(in io.Reader, out io.Writer, cfg config.Config, cwd, rootOverride string) error {
	data, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	var req Request
	if len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &req); err != nil {
			return fmt.Errorf("invalid hook json: %w", err)
		}
	}
	dec, err := Evaluate(req, cfg, cwd, rootOverride)
	if err != nil {
		dec = deny(fmt.Sprintf("rgw-ast hook error: %v", err))
	}
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	return enc.Encode(dec)
}

func allow() Decision {
	return Decision{PermissionDecision: "allow"}
}

func deny(reason string) Decision {
	return Decision{
		PermissionDecision:       "deny",
		PermissionDecisionReason: reason,
		HookSpecificOutput: &hookSpecific{
			HookEventName:            "PreToolUse",
			PermissionDecision:       "deny",
			PermissionDecisionReason: reason,
		},
	}
}

func normalizeTool(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	if i := strings.LastIndex(name, "__"); i >= 0 {
		name = name[i+2:]
	}
	return name
}

func isShellish(name string) bool {
	n := normalizeTool(name)
	switch n {
	case "bash", "shell", "terminal", "runterminalcommand", "run_terminal_command",
		"exec", "execute", "powershell", "cmd", "run":
		return true
	default:
		return strings.Contains(n, "shell") || strings.Contains(n, "bash") || strings.Contains(n, "terminal")
	}
}

func toolName(req Request) string {
	name := strings.ToLower(strings.TrimSpace(req.ToolName))
	if name == "" {
		name = strings.ToLower(strings.TrimSpace(req.Name))
	}
	return name
}

// targetPath returns the absolute file a file tool would change, or "" when
// the input names no path.
func targetPath(input map[string]any, cwd string) string {
	p := firstString(input, "file_path", "notebook_path", "path", "filePath", "target_file")
	if strings.TrimSpace(p) == "" {
		return ""
	}
	return absPath(p, cwd)
}

func absPath(p, cwd string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, p)
	}
	return filepath.Clean(p)
}

// repoFor returns the git worktree that contains path, which may not exist yet.
func repoFor(path string) (string, bool) {
	dir := path
	for {
		if st, err := os.Stat(filepath.Join(dir, ".git")); err == nil && (st.IsDir() || st.Mode().IsRegular()) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// gitIgnored reports whether git ignores path inside repo. Tracked files are
// never reported ignored.
func gitIgnored(repo, path string) bool {
	rel, err := filepath.Rel(repo, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return false
	}
	return exec.Command("git", "-C", repo, "check-ignore", "-q", "--", rel).Run() == nil
}

// judge decides whether a write target is enforced. With a root override
// every target belongs to that root. Otherwise a target outside every git
// worktree, or ignored by its worktree, is free, and the containing worktree's
// measure decides. Each worktree is measured at most once per event.
type judge struct {
	cfg      config.Config
	override string
	measured map[string]bool
}

func (j *judge) enforced(target string) (string, bool, error) {
	repo := j.override
	if repo == "" {
		var ok bool
		if repo, ok = repoFor(target); !ok || gitIgnored(repo, target) {
			return "", false, nil
		}
	}
	if on, ok := j.measured[repo]; ok {
		return repo, on, nil
	}
	m, err := measure.CountCachedOpts(repo, j.cfg, measure.CountOpts{Refresh: true})
	if err != nil {
		return "", false, err
	}
	on := enforce.Decide(j.cfg, m.LOC).Enforced
	j.measured[repo] = on
	return repo, on, nil
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}
