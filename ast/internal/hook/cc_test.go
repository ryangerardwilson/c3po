package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func decision(t *testing.T, cfg config.Config, req Request) (string, string) {
	t.Helper()
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(bytes.NewReader(data), &out, cfg, "/", ""); err != nil {
		t.Fatal(err)
	}
	var dec Decision
	if err := json.Unmarshal(out.Bytes(), &dec); err != nil {
		t.Fatal(err, out.String())
	}
	return dec.PermissionDecision, dec.PermissionDecisionReason
}

func TestMCPEditorToolsAreDenied(t *testing.T) {
	repo, cfg := enforcedRepo(t)
	names := []string{"mcp__ast-mcp__edit_file", "mcp__ast-mcp__file_write", "mcp__ast-mcp__file_patch"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			got, reason := decision(t, cfg, Request{
				Cwd: repo, ToolName: name,
				ToolInput: map[string]any{"file_path": filepath.Join(repo, "a.go")},
			})
			if got != "deny" || !strings.Contains(reason, "c3po ast patch") {
				t.Fatalf("%s: %s %q", name, got, reason)
			}
		})
	}
}

func TestHookDeniesNewFileOverMaxCC(t *testing.T) {
	repo, cfg := enforcedRepo(t)
	cases := []struct{ name, path, body, want string }{
		{"go", "roster.go", highGo(), "Assign"},
		{"py", "roster.py", highPy(), "assign"},
		{"ts", "roster.ts", highTS(), "assign"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, reason := decision(t, cfg, Request{
				Cwd: repo, ToolName: "Write",
				ToolInput: map[string]any{
					"file_path": filepath.Join(repo, "app", c.path),
					"content":   c.body,
				},
			})
			if got != "deny" || !strings.Contains(reason, c.want) || !strings.Contains(reason, "crosses") {
				t.Fatalf("%s: %s %q", c.name, got, reason)
			}
		})
	}
}

func TestHookAllowsNewFileUnderMaxCC(t *testing.T) {
	repo, cfg := enforcedRepo(t)
	body := "package p\n\nfunc Assign(x int) int { return x }\n"
	got, reason := decision(t, cfg, Request{
		Cwd: repo, ToolName: "Write",
		ToolInput: map[string]any{
			"file_path": filepath.Join(repo, "ok.go"),
			"content":   body,
		},
	})
	if got != "allow" {
		t.Fatalf("%s %s", got, reason)
	}
}

func TestHookAllowsHighCCWhenUnenforced(t *testing.T) {
	repo, cfg := enforcedRepo(t)
	cfg.ThresholdLOC = 100000
	got, reason := decision(t, cfg, Request{
		Cwd: repo, ToolName: "Write",
		ToolInput: map[string]any{
			"file_path": filepath.Join(repo, "big.go"),
			"content":   highGo(),
		},
	})
	if got != "allow" {
		t.Fatalf("%s %s", got, reason)
	}
}

func TestHookDeniesUnparsedNewGo(t *testing.T) {
	repo, cfg := enforcedRepo(t)
	got, reason := decision(t, cfg, Request{
		Cwd: repo, ToolName: "Write",
		ToolInput: map[string]any{
			"file_path": filepath.Join(repo, "bad.go"),
			"content":   "package p\n\nfunc F( {\n",
		},
	})
	if got != "deny" || !strings.Contains(reason, "unparsed") {
		t.Fatalf("%s %s", got, reason)
	}
}

func highGo() string {
	var b strings.Builder
	b.WriteString("package p\n\nfunc Assign(x int) int {\n")
	for i := 1; i < 35; i++ {
		fmt.Fprintf(&b, "\tif x == %d {\n\t\treturn %d\n\t}\n", i, i)
	}
	b.WriteString("\treturn 0\n}\n")
	return b.String()
}

func highPy() string {
	var b strings.Builder
	b.WriteString("def assign(x):\n")
	for i := 0; i < 32; i++ {
		fmt.Fprintf(&b, "    if x == %d:\n        return %d\n", i, i)
	}
	b.WriteString("    return 0\n")
	return b.String()
}

func highTS() string {
	var b strings.Builder
	b.WriteString("export function assign(x: number): number {\n")
	for i := 0; i < 32; i++ {
		fmt.Fprintf(&b, "  if (x === %d) { return %d }\n", i, i)
	}
	b.WriteString("  return 0\n}\n")
	return b.String()
}
