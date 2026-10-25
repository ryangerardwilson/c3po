package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func decideReason(t *testing.T, req Request) (string, string) {
	t.Helper()
	repo, cfg := enforcedRepo(t)
	if req.Cwd == "" {
		req.Cwd = repo
	}
	big := strings.Repeat("// line\n", 300)
	if err := os.WriteFile(filepath.Join(repo, "big.go"), []byte("package a\n"+big), 0o644); err != nil {
		t.Fatal(err)
	}
	if fp, ok := req.ToolInput["file_path"].(string); ok && !filepath.IsAbs(fp) && strings.HasPrefix(fp, "@") {
		req.ToolInput["file_path"] = filepath.Join(repo, fp[1:])
	}
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

func TestHookReducesAgentFriction(t *testing.T) {
	sh := func(cmd string) Request { return Request{ToolName: "Bash", ToolInput: map[string]any{"command": cmd}} }
	read := func(path string, limit any) Request {
		in := map[string]any{"file_path": path}
		if limit != nil {
			in["limit"] = limit
		}
		return Request{ToolName: "Read", ToolInput: in}
	}
	cases := []struct {
		name   string
		req    Request
		want   string
		reason string
	}{
		{"long read without limit", read("@big.go", nil), "deny", "c3po ast read big.go --lines 1-200"},
		{"bounded read", read("@big.go", 150.0), "allow", ""},
		{"limit over the cap", read("@big.go", 500.0), "deny", "limit <= 200"},
		{"short file whole", read("@a.go", nil), "allow", ""},
		{"read outside every repository", read(filepath.Join(t.TempDir(), "x"), nil), "allow", ""},
		{"edit carries the hash", Request{ToolName: "Edit", ToolInput: map[string]any{"file_path": "@a.go"}}, "deny", "c3po ast patch a.go --expect-hash "},
		{"write of a missing file", Request{ToolName: "Write", ToolInput: map[string]any{"file_path": "@notes/new.md"}}, "allow", ""},
		{"python inline write", sh(`python3 -c "open('a.go','w').write('x')"`), "deny", "inline interpreter"},
		{"python inline read", sh(`python3 -c "print(open('a.go').read())"`), "allow", ""},
		{"node inline write", sh(`node -e "require('fs').writeFileSync('a.go','x')"`), "deny", "inline interpreter"},
		{"perl inline unlink", sh(`perl -e 'unlink "a.go"'`), "deny", "inline interpreter"},
		{"bare openspec archive", sh("openspec archive x --yes"), "deny", "c3po ast exec -- openspec archive x --yes"},
		{"read-only openspec", sh("openspec validate x --strict"), "allow", ""},
		{"wrapped openspec archive", sh("rgw-ast exec -- openspec archive x"), "allow", ""},
		{"wrapped c3po ast exec", sh("c3po ast exec -- openspec archive x"), "allow", ""},
		{"go generate", sh("go generate ./..."), "deny", "c3po ast exec -- go generate ./..."},
		{"git mv names move", sh("git mv a.go b.go"), "deny", "c3po ast move"},
		{"mv names move", sh("mv a.go b.go"), "deny", "c3po ast move"},
		{"rm names delete", sh("rm a.go"), "deny", "c3po ast delete"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, reason := decideReason(t, c.req)
			if got != c.want || !strings.Contains(reason, c.reason) {
				t.Fatalf("got %s %q, want %s containing %q", got, reason, c.want, c.reason)
			}
		})
	}
}
