package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestHookDeniesMutationWithRgwComment(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n//1\n//2\n//3\n//4\n//5\n//6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.ThresholdLOC = 1
	cfg.Enforcement.Mode = "auto"
	cfg.Cache.Dir = t.TempDir()

	in := bytes.NewBufferString(`{"tool_name":"Bash","tool_input":{"command":"printf x > should-be-denied.txt # rgw-ast"}}`)
	var out bytes.Buffer
	if err := Run(in, &out, cfg, root, root); err != nil {
		t.Fatal(err)
	}
	var dec Decision
	if err := json.Unmarshal(out.Bytes(), &dec); err != nil {
		t.Fatal(err, out.String())
	}
	if dec.PermissionDecision != "deny" {
		t.Fatalf("expected deny, got %s (%s)", dec.PermissionDecision, out.String())
	}
}

func TestHookAllowsLeadingRgwAst(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n//1\n//2\n//3\n//4\n//5\n//6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.ThresholdLOC = 1
	cfg.Cache.Dir = t.TempDir()
	in := bytes.NewBufferString(`{"tool_name":"Bash","tool_input":{"command":"rgw-ast status --json"}}`)
	var out bytes.Buffer
	if err := Run(in, &out, cfg, root, root); err != nil {
		t.Fatal(err)
	}
	var dec Decision
	if err := json.Unmarshal(out.Bytes(), &dec); err != nil {
		t.Fatal(err)
	}
	if dec.PermissionDecision != "allow" {
		t.Fatalf("%s", out.String())
	}
}

func TestHookDeniesCompoundOrEvaluatedRgwAst(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n//1\n//2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.ThresholdLOC = 1
	cfg.Cache.Dir = t.TempDir()

	commands := []string{
		"rgw-ast status; printf x > f",
		"rgw-ast status && rm -rf target",
		"rgw-ast status | tee f",
		"rgw-ast status > f",
		`rgw-ast status "$(printf x > f)"`,
		"rgw-ast status `printf x > f`",
	}
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			data, err := json.Marshal(Request{ToolName: "Bash", ToolInput: map[string]any{"command": command}})
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := Run(bytes.NewReader(data), &out, cfg, root, root); err != nil {
				t.Fatal(err)
			}
			var dec Decision
			if err := json.Unmarshal(out.Bytes(), &dec); err != nil {
				t.Fatal(err)
			}
			if dec.PermissionDecision != "deny" {
				t.Fatalf("expected deny for %q, got %s", command, out.String())
			}
		})
	}
}

func TestHookAllowsQuotedRgwAstArguments(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n//1\n//2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.ThresholdLOC = 1
	cfg.Cache.Dir = t.TempDir()
	command := `rgw-ast patch f --old 'a;b' --new "c && d"`
	data, err := json.Marshal(Request{ToolName: "Bash", ToolInput: map[string]any{"command": command}})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(bytes.NewReader(data), &out, cfg, root, root); err != nil {
		t.Fatal(err)
	}
	var dec Decision
	if err := json.Unmarshal(out.Bytes(), &dec); err != nil {
		t.Fatal(err)
	}
	if dec.PermissionDecision != "allow" {
		t.Fatalf("expected allow for quoted content, got %s", out.String())
	}
}

func TestHookRoutesDirectDeletionToGuardedCommand(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n//1\n//2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.ThresholdLOC = 1
	cfg.Enforcement.Mode = "auto"
	cfg.Cache.Dir = t.TempDir()
	data, err := json.Marshal(Request{ToolName: "Delete", ToolInput: map[string]any{"path": "a.go"}})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(bytes.NewReader(data), &out, cfg, root, root); err != nil {
		t.Fatal(err)
	}
	var dec Decision
	if err := json.Unmarshal(out.Bytes(), &dec); err != nil {
		t.Fatal(err)
	}
	if dec.PermissionDecision != "deny" || !strings.Contains(dec.PermissionDecisionReason, "c3po ast delete a.go --expect-hash ") {
		t.Fatalf("unexpected decision %s", out.String())
	}
}
