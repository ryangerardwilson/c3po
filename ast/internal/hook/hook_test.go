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

// measureCacheFile returns the hook's measure cache entry, or "".
func measureCacheFile(t *testing.T, cfg config.Config) string {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(cfg.Cache.Dir, "measure", "*.json"))
	if len(matches) > 1 {
		t.Fatalf("cache entries %v", matches)
	}
	if len(matches) == 0 {
		return ""
	}
	return matches[0]
}

func setCachedLOC(t *testing.T, path string, loc int) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var ent map[string]any
	if err := json.Unmarshal(data, &ent); err != nil {
		t.Fatal(err)
	}
	ent["loc"] = loc
	data, _ = json.Marshal(ent)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHookReusesMeasureUntilTheTreeChanges(t *testing.T) {
	repo, cfg := enforcedRepo(t)
	cfg.ThresholdLOC = 50
	edit := Request{ToolName: "Edit", ToolInput: map[string]any{"file_path": filepath.Join(repo, "a.go")}}

	create := Request{ToolName: "Write", ToolInput: map[string]any{"file_path": filepath.Join(repo, "new.go")}}
	if got := decide(t, cfg, create); got != "allow" {
		t.Fatalf("create %s", got)
	}
	if path := measureCacheFile(t, cfg); path != "" {
		t.Fatalf("a create measured the tree: %s", path)
	}

	if got := decide(t, cfg, edit); got != "allow" {
		t.Fatalf("small tree %s", got)
	}
	path := measureCacheFile(t, cfg)
	if path == "" {
		t.Fatal("the judged edit stored no measure")
	}
	// A matching fingerprint is trusted: a cached count over the threshold
	// enforces although the tree itself is small.
	setCachedLOC(t, path, 1000)
	if got := decide(t, cfg, edit); got != "deny" {
		t.Fatalf("the cached measure was recounted: %s", got)
	}

	// With a small cached count, growing the tree past the threshold
	// changes the fingerprint and the hook recounts.
	setCachedLOC(t, path, 3)
	if got := decide(t, cfg, edit); got != "allow" {
		t.Fatalf("small cached count %s", got)
	}
	if err := os.WriteFile(filepath.Join(repo, "big.go"), []byte("package a\n"+strings.Repeat("//\n", 100)), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := decide(t, cfg, edit); got != "deny" {
		t.Fatalf("a stale cache hid a tree over the threshold: %s", got)
	}
}

func TestHookReasonAvoidsAnAmbiguousRelativePath(t *testing.T) {
	repo, cfg := enforcedRepo(t)
	sub := filepath.Join(repo, "ast")
	for _, dir := range []string{repo, sub} {
		if err := os.MkdirAll(filepath.Join(dir, "internal"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "internal", "v.go"), []byte("package v\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(repo, "internal", "v.go")
	data, _ := json.Marshal(Request{Cwd: sub, ToolName: "Edit", ToolInput: map[string]any{"file_path": target}})
	var out bytes.Buffer
	if err := Run(bytes.NewReader(data), &out, cfg, "/", ""); err != nil {
		t.Fatal(err)
	}
	var dec Decision
	if err := json.Unmarshal(out.Bytes(), &dec); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dec.PermissionDecisionReason, "c3po ast patch "+target+" ") {
		t.Fatalf("reason %q", dec.PermissionDecisionReason)
	}
}
