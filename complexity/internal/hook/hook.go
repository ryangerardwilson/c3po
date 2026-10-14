package hook

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/ryangerardwilson/c3po/complexity/internal/agents"
	"github.com/ryangerardwilson/c3po/complexity/internal/analyze"
	"github.com/ryangerardwilson/c3po/complexity/internal/config"
)

type event struct {
	HookEventName   string          `json:"hook_event_name"`
	HookEventName2  string          `json:"hookEventName"`
	ToolName        string          `json:"tool_name"`
	ToolName2       string          `json:"toolName"`
	ToolInput       json.RawMessage `json:"tool_input"`
	ToolInput2      json.RawMessage `json:"toolInput"`
	Cwd             string          `json:"cwd"`
	WorkspaceRoot   string          `json:"workspaceRoot"`
	Reason          string          `json:"reason"`
	SessionID       string          `json:"session_id"`
	SessionID2      string          `json:"sessionId"`
	StopHookActive  bool            `json:"stop_hook_active"`
	StopHookActive2 bool            `json:"stopHookActive"`
}

// Run reads one hook event and writes an allow, deny, or block decision.
// It always returns 0. Unjudgeable events are allowed.
func Run(r io.Reader, w io.Writer) int {
	var ev event
	if err := json.NewDecoder(r).Decode(&ev); err != nil {
		allow(w)
		return 0
	}
	cwd := ev.Cwd
	if cwd == "" {
		cwd = ev.WorkspaceRoot
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	cfg, err := config.Merge(cwd, "")
	if err != nil || cfg.Enforcement.Mode == "off" {
		allow(w)
		return 0
	}
	remember(cwd, ev)
	if isStop(ev) {
		stop(w, cwd, cfg, ev)
		return 0
	}
	allow(w)
	return 0
}

func isStop(ev event) bool {
	return strings.EqualFold(ev.name(), "Stop")
}

func (ev event) name() string {
	if ev.HookEventName != "" {
		return ev.HookEventName
	}
	return ev.HookEventName2
}

func (ev event) tool() string {
	if ev.ToolName != "" {
		return ev.ToolName
	}
	return ev.ToolName2
}

func (ev event) input() map[string]any {
	raw := ev.ToolInput
	if len(raw) == 0 {
		raw = ev.ToolInput2
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return m
}

func stop(w io.Writer, cwd string, cfg config.Config, ev event) {
	if skipStop(cfg, ev) {
		allow(w)
		return
	}
	root, ok := config.GitRoot(cwd)
	if !ok {
		allow(w)
		return
	}
	diff, paths := stopScope(cwd, root, ev)
	res, err := analyze.Run(analyze.Options{
		Paths: paths, Config: cfg, Diff: diff, Ratchet: true, WantModels: true,
	})
	if err != nil || !res.Violations {
		allow(w)
		return
	}
	block(w, stopReason(cwd, res))
}

func stopReason(cwd string, res analyze.Result) string {
	shown := res.ForDisplayAt(cwd)
	var b strings.Builder
	for _, line := range shown.Report.Lines() {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if b.Len() == 0 {
		b.WriteString("complexity ratchet found violations\n")
	}
	b.WriteString("Run ")
	b.WriteString(agents.Recheck)
	return b.String()
}

func closed(reason string) bool {
	switch strings.ToLower(reason) {
	case "channel_closed", "shutdown":
		return true
	default:
		return false
	}
}

func field(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok {
			return s
		}
	}
	return ""
}

func allow(w io.Writer) {
	write(w, map[string]string{"decision": "allow", "permissionDecision": "allow"})
}

func block(w io.Writer, reason string) {
	write(w, map[string]string{"decision": "block", "reason": reason})
}

func write(w io.Writer, v any) {
	buf, err := json.Marshal(v)
	if err != nil {
		return
	}
	buf = append(buf, '\n')
	_, _ = io.Copy(w, bytes.NewReader(buf))
}
