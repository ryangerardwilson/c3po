package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyAllRollsBackNothingOnBadHash(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.go")
	b := filepath.Join(root, "b.go")
	if err := os.WriteFile(a, []byte("package p\nfunc A() { keep }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("package p\nfunc B() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ha, err := HashFile(a)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ApplyAll(root, []FileChange{
		{Path: "a.go", ExpectHash: ha, Old: "keep", New: "changed"},
		{Path: "b.go", ExpectHash: "deadbeef", Old: "func", New: "func"},
	})
	if err == nil {
		t.Fatal("expected hash failure")
	}
	got, _ := os.ReadFile(a)
	if strings.Contains(string(got), "changed") {
		t.Fatalf("file one was written: %s", got)
	}
}

func TestApplyAllFormatsGo(t *testing.T) {
	root := t.TempDir()
	body := "package p\nfunc A(){println( 1 )}\n"
	path := filepath.Join(root, "a.go")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	res, err := ApplyAll(root, []FileChange{{
		Path: "a.go", ExpectHash: sum, Symbol: "A", Old: "println( 1 )", New: "println(2)",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Formatter != "gofmt" {
		t.Fatalf("formatter %q", res[0].Formatter)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "println(2)") {
		t.Fatalf("body %s", got)
	}
}
