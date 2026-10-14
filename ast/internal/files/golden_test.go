package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoldenSymbolPatchStaysInsideMultilineMethod(t *testing.T) {
	root := t.TempDir()
	body := `class PromiseEngine:
    def other(self) -> str:
        return current_runtime_policy()

    def promise(
        self,
        policy: str,
    ) -> str:
        return current_runtime_policy()
`
	path := filepath.Join(root, "engine.py")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	res, err := DryRun(root, []FileChange{{
		Path: "engine.py", ExpectHash: sum, Symbol: "PromiseEngine.promise",
		Old: "current_runtime_policy()", New: "current_runtime_policy(1)",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || !strings.Contains(res[0].Diff, "current_runtime_policy(1)") {
		t.Fatalf("dry run %+v", res)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatal("dry run wrote the file")
	}
	applied, err := ApplyAll(root, []FileChange{{
		Path: "engine.py", ExpectHash: sum, Symbol: "PromiseEngine.promise",
		Old: "current_runtime_policy()", New: "current_runtime_policy(1)",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if applied[0].Hash == "" {
		t.Fatal("missing hash")
	}
	got, _ = os.ReadFile(path)
	text := string(got)
	if strings.Count(text, "current_runtime_policy(1)") != 1 {
		t.Fatalf("edit escaped or missed:\n%s", text)
	}
	if !strings.Contains(text, "def other") || !strings.Contains(text, "return current_runtime_policy()\n") {
		t.Fatalf("other method changed:\n%s", text)
	}
}

func TestGoldenApplyWriteFailureRollsBack(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.go")
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	b := filepath.Join(sub, "b.go")
	if err := os.WriteFile(a, []byte("package p\nfunc A() { keep }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("package p\nfunc B() { keep }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ha, _ := HashFile(a)
	hb, _ := HashFile(b)
	if err := os.Chmod(sub, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(sub, 0o755)
	_, err := ApplyAll(root, []FileChange{
		{Path: "a.go", ExpectHash: ha, Old: "keep", New: "changed"},
		{Path: "sub/b.go", ExpectHash: hb, Old: "keep", New: "changed"},
	})
	if err == nil {
		t.Fatal("expected write failure")
	}
	got, _ := os.ReadFile(a)
	if strings.Contains(string(got), "changed") {
		t.Fatalf("rollback missed: %s", got)
	}
}
