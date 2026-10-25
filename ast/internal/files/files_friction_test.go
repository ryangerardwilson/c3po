package files

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadHeaderAndStrictEnd(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "f.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("x\n", 10)), 0o644); err != nil {
		t.Fatal(err)
	}
	text, next, err := FormatReadRange(path, "f.txt", 1, 8, 5, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if next != 6 || !strings.HasPrefix(text, "# f.txt lines 1-5 of 10 (capped at 5; next: --lines 6-10)\n") {
		t.Fatalf("header %q next %d", strings.SplitN(text, "\n", 2)[0], next)
	}
	if _, _, err := FormatReadRange(path, "f.txt", 8, 12, 5, false, true); err == nil {
		t.Fatal("strict range past end of file should fail")
	}
	if _, _, err := FormatReadRange(path, "f.txt", 8, 12, 5, false, false); err != nil {
		t.Fatalf("non-strict range past end should clamp: %v", err)
	}
}

func TestPatchReportsLinesAndReplacesAll(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "m.py")
	if err := os.WriteFile(path, []byte("x = 1\ny = 2\n\nx = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, _ := HashFile(path)
	_, err := PatchOps(path, sum, []Op{{Old: "x = 1", New: "x = 2"}})
	if err == nil || !strings.Contains(err.Error(), "lines 1, 4") || !strings.Contains(err.Error(), "--all") {
		t.Fatalf("ambiguous error %v", err)
	}
	if _, err := PatchOps(path, "deadbeef", []Op{{Old: "y = 2", New: "y = 3"}}); err == nil || !strings.Contains(err.Error(), "c3po ast hash") {
		t.Fatalf("stale error %v", err)
	}
	if _, err := PatchOps(path, sum, []Op{{Old: "x = 1", New: "x = 2", All: true}}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if strings.Count(string(got), "x = 2") != 2 {
		t.Fatalf("replace all: %s", got)
	}
}

func TestMoveExact(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git unavailable: %v %s", err, out)
	}
	a := filepath.Join(root, "a.go")
	if err := os.WriteFile(a, []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", root, "add", "a.go").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	sum, _ := HashFile(a)
	if _, _, err := MoveExact(root, "a.go", "sub/b.go", sum, false); err == nil {
		t.Fatal("missing destination directory without --parents should fail")
	}
	if _, _, err := MoveExact(root, "a.go", "b.go", "deadbeef", false); err == nil {
		t.Fatal("stale hash should fail")
	}
	h, viaGit, err := MoveExact(root, "a.go", "sub/b.go", sum, true)
	if err != nil {
		t.Fatal(err)
	}
	if h != sum || !viaGit {
		t.Fatalf("hash %s git %v", h, viaGit)
	}
	if _, err := os.Stat(a); !os.IsNotExist(err) {
		t.Fatal("source still exists")
	}
	staged, _ := exec.Command("git", "-C", root, "diff", "--cached", "--name-only").Output()
	if !strings.Contains(string(staged), "sub/b.go") {
		t.Fatalf("move not staged: %s", staged)
	}
	c := filepath.Join(root, "c.txt")
	if err := os.WriteFile(c, []byte("c\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "d.txt"), []byte("d\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	csum, _ := HashFile(c)
	if _, _, err := MoveExact(root, "c.txt", "d.txt", csum, false); err == nil {
		t.Fatal("existing destination should fail")
	}
	if _, viaGit, err := MoveExact(root, "c.txt", "e.txt", csum, false); err != nil || viaGit {
		t.Fatalf("untracked move err=%v git=%v", err, viaGit)
	}
}

func TestApplyMultiHunkDiffThatChangesLineCount(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "f.txt")
	if err := os.WriteFile(path, []byte("a\nb\nc\nd\ne\nf\ng\nh\ni\nj\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diff := "--- a/f.txt\n+++ b/f.txt\n@@ -1,3 +1,1 @@\n-a\n-b\n c\n@@ -8,3 +6,4 @@\n h\n-i\n+I\n+I2\n j\n"
	changes, err := ParseApplyBytes([]byte(diff))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyAllOpts(root, changes, ApplyOpts{NoFormat: true}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "c\nd\ne\nf\ng\nh\nI\nI2\nj\n" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyPlainDiffAndMove(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a.txt", "one\ntwo\nthree\n")
	write("gone.txt", "bye\n")
	write("m.txt", "move me\n")
	diff := `diff --git a/a.txt b/a.txt
index 1111111..2222222 100644
--- a/a.txt
+++ b/a.txt
@@ -1,3 +1,3 @@
 one
-two
+TWO
 three
diff --git a/new.txt b/new.txt
new file mode 100644
--- /dev/null
+++ b/new.txt
@@ -0,0 +1 @@
+fresh
diff --git a/gone.txt b/gone.txt
deleted file mode 100644
--- a/gone.txt
+++ /dev/null
@@ -1 +0,0 @@
-bye
`
	changes, err := ParseApplyBytes([]byte(diff))
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 3 || changes[1].Op != "create" || changes[2].Op != "delete" || changes[2].Path != "gone.txt" {
		t.Fatalf("parsed %+v", changes)
	}
	msum, _ := HashFile(filepath.Join(root, "m.txt"))
	changes = append(changes, FileChange{Op: "move", Path: "m.txt", To: "moved.txt", ExpectHash: msum})
	res, err := ApplyAllOpts(root, changes, ApplyOpts{NoFormat: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 4 || res[3].To != "moved.txt" {
		t.Fatalf("results %+v", res)
	}
	got, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	fresh, _ := os.ReadFile(filepath.Join(root, "new.txt"))
	moved, _ := os.ReadFile(filepath.Join(root, "moved.txt"))
	if string(got) != "one\nTWO\nthree\n" || string(fresh) != "fresh\n" || string(moved) != "move me\n" {
		t.Fatalf("a=%q new=%q moved=%q", got, fresh, moved)
	}
	if _, err := os.Stat(filepath.Join(root, "gone.txt")); !os.IsNotExist(err) {
		t.Fatal("gone.txt not deleted")
	}

	// The same diff again: context no longer matches, nothing changes.
	write("a.txt", "one\nTWO\nthree\n")
	stale, err := ParseApplyBytes([]byte(diff))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyAll(root, stale[:1]); err == nil {
		t.Fatal("stale context should fail")
	}
	if _, err := ParseApplyBytes([]byte("--- a/x\n+++ b/y\n@@ -1 +1 @@\n-a\n+b\n")); err == nil {
		t.Fatal("rename diff should be refused")
	}
	gofmtDiff, err := ParseApplyBytes([]byte("diff -u a.go.orig a.go\n--- a.go.orig\n+++ a.go\n@@ -1 +1 @@\n-a\n+b\n"))
	if err != nil || len(gofmtDiff) != 1 || gofmtDiff[0].Path != "a.go" || gofmtDiff[0].Op != "edit" {
		t.Fatalf("gofmt -d diff %+v %v", gofmtDiff, err)
	}
}
