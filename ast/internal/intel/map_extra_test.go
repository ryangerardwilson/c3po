package intel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestTOMLArrayValuesAreNotKeys(t *testing.T) {
	root := t.TempDir()
	body := "[project]\nname = \"demo\"\ndependencies = [\n  \"ruff>=0.1\",\n]\n"
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := Map(root, "pyproject.toml", config.Default(), 50)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Kind == "key" && strings.Contains(e.Name, "ruff") {
			t.Fatalf("array value reported as key: %+v", e)
		}
		if e.Kind == "value" && e.Name == "ruff>=0.1" {
			return
		}
	}
	t.Fatalf("missing array value %#v", entries)
}

func TestWorkflowJobs(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "name: CI\non: push\njobs:\n  lint:\n    runs-on: ubuntu-latest\n    steps:\n      - name: Lint\n        run: echo lint\n  test:\n    needs: [lint]\n    if: github.ref == 'refs/heads/main'\n    runs-on: ubuntu-latest\n    steps:\n      - name: Run tests\n        run: go test\n"
	if err := os.WriteFile(filepath.Join(dir, "ci.yml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := Map(root, ".github/workflows/ci.yml", config.Default(), 50)
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, e := range entries {
		kinds = append(kinds, e.Kind+":"+e.Name)
	}
	joined := strings.Join(kinds, " ")
	if !strings.Contains(joined, "workflow:CI") || !strings.Contains(joined, "job:test") || !strings.Contains(joined, "step:test/Run tests") || !strings.Contains(joined, "dependency:test->lint") || !strings.Contains(joined, "condition:test") {
		t.Fatalf("%v", kinds)
	}
}

func TestMapDepthAndOffset(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package p\nfunc A() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "b.go"), []byte("package p\nfunc B() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	top, err := MapQuery(root, "", cfg, MapOptions{Limit: 20, Depth: 0})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range top.Entries {
		if strings.Contains(e.Path, "sub/") {
			t.Fatalf("depth 0 entered sub: %+v", e)
		}
	}
	page, err := MapQuery(root, "", cfg, MapOptions{Limit: 1, Offset: 0, Depth: -1})
	if err != nil {
		t.Fatal(err)
	}
	if !page.Truncated || page.NextOffset == nil || *page.NextOffset != 1 || len(page.Entries) != 1 {
		t.Fatalf("page %+v", page)
	}
	next, err := MapQuery(root, "", cfg, MapOptions{Limit: 50, Offset: *page.NextOffset, Depth: -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Entries) == 0 || next.Entries[0].Path == page.Entries[0].Path && next.Entries[0].Line == page.Entries[0].Line && next.Entries[0].Name == page.Entries[0].Name {
		t.Fatalf("continuation repeated %+v / %+v", page.Entries, next.Entries)
	}
}
