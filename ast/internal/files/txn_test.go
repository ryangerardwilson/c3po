package files

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

// failingTxn deletes an executable and then edits a file in a read-only
// directory, so the delete commits and the edit fails.
func failingTxn(t *testing.T) (root, exe string, body []byte) {
	t.Helper()
	root = t.TempDir()
	exe = filepath.Join(root, "run.sh")
	body = []byte("#!/bin/sh\necho hi\n")
	if err := os.WriteFile(exe, body, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(exe, 0o755); err != nil {
		t.Fatal(err)
	}
	ro := filepath.Join(root, "ro")
	if err := os.Mkdir(ro, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ro, "b.txt"), []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ro, 0o755) })
	if f, err := os.CreateTemp(ro, "probe"); err == nil {
		f.Close()
		t.Skip("read-only directory is writable (running as root?)")
	}
	return root, exe, body
}

func applyFailing(t *testing.T, root, exe string) {
	t.Helper()
	he, _ := HashFile(exe)
	hb, _ := HashFile(filepath.Join(root, "ro", "b.txt"))
	_, err := ApplyAllOpts(root, []FileChange{
		{Path: "run.sh", Op: "delete", ExpectHash: he},
		{Path: "ro/b.txt", ExpectHash: hb, Old: "old", New: "new"},
	}, ApplyOpts{NoFormat: true})
	if err == nil {
		t.Fatal("expected the edit in a read-only directory to fail")
	}
}

func assertRestored(t *testing.T, root, exe string, body []byte) {
	t.Helper()
	info, err := os.Stat(exe)
	if err != nil {
		t.Fatalf("deleted file not restored: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode %v, want 0755", info.Mode().Perm())
	}
	if got, _ := os.ReadFile(exe); string(got) != string(body) {
		t.Fatalf("body %q", got)
	}
	assertNoKeep(t, root)
}

func assertNoKeep(t *testing.T, dir string) {
	t.Helper()
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), keepPrefix) {
			t.Fatalf("left %s behind", e.Name())
		}
	}
}

func TestFailedApplyRestoresDeletedExecutable(t *testing.T) {
	root, exe, body := failingTxn(t)
	applyFailing(t, root, exe)
	assertRestored(t, root, exe, body)
}

func TestFailedApplyRestoresDeletedExecutableWithoutLink(t *testing.T) {
	root, exe, body := failingTxn(t)
	linkFile = func(string, string) error {
		return &os.LinkError{Op: "link", Err: syscall.EPERM}
	}
	t.Cleanup(func() { linkFile = os.Link })
	applyFailing(t, root, exe)
	assertRestored(t, root, exe, body)
}

func TestApplyDeleteLeavesNoKeep(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.txt")
	if err := os.WriteFile(a, []byte("gone\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ha, _ := HashFile(a)
	res, err := ApplyAll(root, []FileChange{{Path: "a.txt", Op: "delete", ExpectHash: ha}})
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Hash != ha {
		t.Fatalf("hash %s, want %s", res[0].Hash, ha)
	}
	if _, err := os.Lstat(a); !os.IsNotExist(err) {
		t.Fatalf("file still present: %v", err)
	}
	assertNoKeep(t, root)
}

func TestRestoreDeletedKeepsSiblingWhenOccupied(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.txt")
	if err := os.WriteFile(a, []byte("orig\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	keep, err := stageDelete(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a, []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := restoreDeleted(a, keep); err == nil || !strings.Contains(err.Error(), keep) {
		t.Fatalf("want occupied error naming %s, got %v", keep, err)
	}
	if got, _ := os.ReadFile(a); string(got) != "other\n" {
		t.Fatalf("occupied path overwritten: %q", got)
	}
	if got, _ := os.ReadFile(keep); string(got) != "orig\n" {
		t.Fatalf("sibling %q", got)
	}
}

func TestDryRunDeleteHasNoDiff(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.txt")
	if err := os.WriteFile(a, []byte("secret body\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ha, _ := HashFile(a)
	res, err := DryRun(root, []FileChange{{Path: "a.txt", Op: "delete", ExpectHash: ha}})
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Op != "delete" || res[0].Diff != "" {
		t.Fatalf("result %+v", res[0])
	}
	if _, err := os.Stat(a); err != nil {
		t.Fatalf("dry run touched the file: %v", err)
	}
}

func TestApplyMoveReportsPrepareHash(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.txt")
	if err := os.WriteFile(a, []byte("move me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ha, _ := HashFile(a)
	res, err := ApplyAll(root, []FileChange{{Path: "a.txt", Op: "move", To: "b.txt", ExpectHash: ha}})
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Hash != ha {
		t.Fatalf("hash %s, want %s", res[0].Hash, ha)
	}
}
