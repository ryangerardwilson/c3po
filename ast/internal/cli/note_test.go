package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/files"
)

func TestNoteWrittenPrintsCx(t *testing.T) {
	bin := stubCx(t, "#!/bin/sh\nprintf 'handle (new) cognitive 25 > 15. Get it under 15.\\n'\n")
	t.Setenv("PATH", filepath.Dir(bin))
	var out bytes.Buffer
	Runner{Out: &out}.noteWritten([]string{"/tmp/a.go"})
	if !bytes.Contains(out.Bytes(), []byte("Get it under 15")) {
		t.Fatal(out.String())
	}
}

func TestNoteWrittenIgnoresFailure(t *testing.T) {
	bin := stubCx(t, "#!/bin/sh\nexit 1\n")
	t.Setenv("PATH", filepath.Dir(bin))
	var out bytes.Buffer
	Runner{Out: &out}.noteWritten([]string{"/tmp/a.go"})
	if out.Len() != 0 {
		t.Fatal(out.String())
	}
}

func TestNoteWrittenSkipsMarkdown(t *testing.T) {
	bin := stubCx(t, "#!/bin/sh\nprintf 'should not run\\n'\n")
	t.Setenv("PATH", filepath.Dir(bin))
	var out bytes.Buffer
	Runner{Out: &out}.noteWritten([]string{"/tmp/a.md"})
	if out.Len() != 0 {
		t.Fatal(out.String())
	}
}

func TestApplyAbsSkipsDeleteAndMove(t *testing.T) {
	got := applyAbs("/r", []files.ApplyResult{
		{Path: "a.go", Op: "edit"},
		{Path: "b.go", Op: "delete"},
		{Path: "c.go", Op: "move"},
	})
	if len(got) != 1 || got[0] != "/r/a.go" {
		t.Fatalf("%v", got)
	}
}

func TestPatchAppendsTheNote(t *testing.T) {
	ws := t.TempDir()
	mustWrite(t, filepath.Join(ws, "f.go"), "package p\nconst X = 1\n")
	bin := stubCx(t, "#!/bin/sh\nprintf 'f (new) cognitive 2 > 1. Get it under 1.\\n'\n")
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, errB bytes.Buffer
	r := NewRunner(&out, &errB)
	r.Cwd = ws
	if code := r.Run([]string{"hash", "f.go"}); code != 0 {
		t.Fatal(code, errB.String())
	}
	hash := firstField(out.String())
	out.Reset()
	if code := r.Run([]string{"patch", "f.go", "--expect-hash", hash, "--old", "X = 1", "--new", "X = 2"}); code != 0 {
		t.Fatal(code, errB.String(), out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("ok  ")) || !bytes.Contains(out.Bytes(), []byte("Get it under 1")) {
		t.Fatalf("%s", out.String())
	}
}

func stubCx(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cx")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func firstField(s string) string {
	for i, c := range s {
		if c == ' ' || c == '\n' {
			return s[:i]
		}
	}
	return s
}
