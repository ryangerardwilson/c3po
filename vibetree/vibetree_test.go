package vibetree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestReportKeepsTheHeavyBranch(t *testing.T) {
	root := build([]item{
		{path: "a.go", size: 10},
		{path: "b/c.go", size: 100},
		{path: "b/d.go", size: 10},
	})
	if root.size != 120 || child(root, "b").size != 110 {
		t.Fatalf("sums %+v", root)
	}
	out := render(root, 50)
	for _, s := range []string{
		"mode loc\n",
		"modules 5\n",
		"p50 100\n",
		".  120\n",
		"  b  110\n",
		"    c.go  100\n",
	} {
		if !strings.Contains(out, s) {
			t.Errorf("missing %q in\n%s", s, out)
		}
	}
	if strings.Contains(out, "a.go") || strings.Contains(out, "d.go") {
		t.Errorf("light files in\n%s", out)
	}
	if strings.Contains(out, "p90 ") || strings.Contains(out, "p95 ") || strings.Contains(out, "p99 ") {
		t.Errorf("other percentiles in\n%s", out)
	}
	p90 := render(root, 90)
	if strings.Contains(p90, "c.go") || !strings.Contains(p90, ".  120\n") {
		t.Errorf("p90\n%s", p90)
	}
}

func TestEmptyTree(t *testing.T) {
	out := render(build(nil), 90)
	if !strings.Contains(out, "modules 0\n") || !strings.Contains(out, "No modules.\n") {
		t.Fatalf("%s", out)
	}
}

func TestWalkCountsTextAndSkipsTheRest(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "notes.txt", "one\ntwo\n")
	write(t, dir, "src/main.go", "package main\n\nfunc main() {}\n")
	write(t, dir, "pic.bin", "a\x00b\n")
	write(t, dir, "node_modules/pkg/index.js", "lots\nof\nlines\n")
	write(t, dir, "empty.txt", "")
	items, err := walkCollect(dir, fileLines)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, it := range items {
		got[it.path] = it.size
	}
	if got["notes.txt"] != 2 || got["src/main.go"] != 3 {
		t.Fatalf("%v", got)
	}
	if _, ok := got["pic.bin"]; ok {
		t.Fatalf("binary counted %v", got)
	}
	if _, ok := got["node_modules/pkg/index.js"]; ok {
		t.Fatalf("dependency counted %v", got)
	}
}

func TestGitHonorsIgnore(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init")
	write(t, dir, ".gitignore", "secret/\n")
	write(t, dir, "keep.txt", "a\nb\n")
	write(t, dir, "secret/hidden.txt", "hide\nme\nplease\n")
	items, ok := gitItems(dir, fileLines)
	if !ok {
		t.Fatal("git list")
	}
	if len(items) != 2 {
		t.Fatalf("%+v", items)
	}
	for _, it := range items {
		if strings.HasPrefix(it.path, "secret") {
			t.Fatalf("ignored %+v", items)
		}
	}
}

func TestRunPercentileAndFlags(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.txt", "one\n")
	write(t, dir, "b/c.txt", strings.Repeat("x\n", 40))
	out, code := run(dir, "-p", "50")
	if code != 0 || !strings.Contains(out, "mode loc\n") || !strings.Contains(out, "\np50 ") || !strings.Contains(out, "c.txt  40\n") {
		t.Fatalf("code %d\n%s", code, out)
	}
	if _, code := run(dir, "-p", "100"); code != 2 {
		t.Fatalf("p100 code %d", code)
	}
	if _, code := run(dir, "--bogus"); code != 2 {
		t.Fatalf("flag code %d", code)
	}
	help, code := run(dir, "help")
	if code != 0 || !strings.Contains(help, "-p N") || !strings.Contains(help, "-m loc|cyclo") || !strings.Contains(help, "--json") || !strings.Contains(help, "to 95") {
		t.Fatalf("help %d\n%s", code, help)
	}
	missing, code := run(dir, filepath.Join(dir, "missing"))
	if code != 1 || !strings.Contains(missing, "vibetree:") {
		t.Fatalf("missing %d\n%s", code, missing)
	}
}

func TestDefaultIsP95Loc(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.txt", "one\n")
	write(t, dir, "b/c.txt", strings.Repeat("x\n", 40))
	out, code := run(dir)
	if code != 0 || !strings.HasPrefix(out, "mode loc\n") || !strings.Contains(out, "\np95 41\n") {
		t.Fatalf("code %d\n%s", code, out)
	}
	if strings.Contains(out, "p50 ") || strings.Contains(out, "p90 ") || strings.Contains(out, "p99 ") {
		t.Fatalf("other percentiles\n%s", out)
	}
	if strings.Contains(out, "c.txt") || strings.Contains(out, "a.txt") {
		t.Fatalf("below the default cut\n%s", out)
	}
	const note = "p95 of 4 modules is rank 4, the largest module. The root is that size because a directory sums its files. Try a lower -p."
	if !strings.Contains(out, "\n\n"+note+"\n") {
		t.Fatalf("note\n%s", out)
	}
	colored, code := run(dir, "--color=always")
	if code != 0 || !strings.Contains(colored, "\x1b[0m\n\n"+note+"\n") {
		t.Fatalf("note colored %d\n%s", code, colored)
	}
	js, code := run(dir, "--json")
	if code != 0 {
		t.Fatalf("json %d\n%s", code, js)
	}
	var rep jsonReport
	if err := json.Unmarshal([]byte(js), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Note != note || rep.Tree == nil || len(rep.Tree.Children) != 0 {
		t.Fatalf("%+v", rep)
	}
}

func TestDirThenPercentileAndColor(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "focus/small/a.txt", strings.Repeat("a\n", 250))
	write(t, dir, "focus/big/b.txt", strings.Repeat("b\n", 5000))
	write(t, dir, "other/c.txt", strings.Repeat("c\n", 5000))
	focus := filepath.Join(dir, "focus")
	out, code := run(dir, focus, "-p", "1", "--color=always")
	if code != 0 || strings.Contains(out, "other") || strings.Contains(out, "c.txt") {
		t.Fatalf("code %d\n%s", code, out)
	}
	if !strings.Contains(out, "small  250") || !strings.Contains(out, "big  5000") {
		t.Fatalf("dir tree\n%s", out)
	}
	if intensityOf(t, out, "small  250") >= intensityOf(t, out, "big  5000") {
		t.Fatalf("intensity small %d big %d\n%s", intensityOf(t, out, "small  250"), intensityOf(t, out, "big  5000"), out)
	}
	plain, code := run(dir, focus, "-p", "1", "--no-color")
	if code != 0 || strings.Contains(plain, "\x1b") {
		t.Fatalf("plain %d\n%s", code, plain)
	}
	if _, code := run(dir, focus, "--color", "loud"); code != 2 {
		t.Fatalf("color flag %d", code)
	}
	if intensity(100, 5000, lineFloor) != minIntensity || intensity(100, 150, lineFloor) != neutralIntensity {
		t.Fatalf("floor %d %d", intensity(100, 5000, lineFloor), intensity(100, 150, lineFloor))
	}
}

func TestCycloSumsFilesInOneDirectory(t *testing.T) {
	dir := t.TempDir()
	files := []struct{ rel, body string }{
		{"lib/a.go", goBranch},
		{"lib/b.py", pyBranch},
		{"lib/notes.txt", "plain\n"},
	}
	for _, f := range files {
		write(t, dir, f.rel, f.body)
	}
	out, code := run(dir, "-m", "cyclo", "-p", "1")
	if code != 0 || strings.Contains(out, "notes.txt") || !strings.HasPrefix(out, "mode cyclo\n") {
		t.Fatalf("code %d\n%s", code, out)
	}
	sum := 0
	for _, f := range files[:2] {
		n, ok := cyclomaticFile(f.rel, []byte(f.body))
		name := filepath.Base(f.rel)
		if !ok || !strings.Contains(out, fmt.Sprintf("%s  %d\n", name, n)) {
			t.Fatalf("%s %d %v\n%s", f.rel, n, ok, out)
		}
		sum += n
	}
	if !strings.Contains(out, fmt.Sprintf("lib  %d\n", sum)) || !strings.Contains(out, fmt.Sprintf(".  %d\n", sum)) {
		t.Fatalf("sum %d\n%s", sum, out)
	}
	loc, code := run(dir, "-m", "loc", "-p", "1")
	if code != 0 || !strings.Contains(loc, "notes.txt") || !strings.HasPrefix(loc, "mode loc\n") {
		t.Fatalf("loc %d\n%s", code, loc)
	}
}

func TestCycloColorDimsTheStraightFunction(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "straight.go", "package p\n\nfunc B() int {\n\treturn 1\n}\n")
	write(t, dir, "branch.go", "package p\n\nfunc A(x int) int {\n\tif x > 0 {\n\t\treturn 1\n\t}\n\treturn 0\n}\n")
	out, code := run(dir, "-m", "cyclo", "-p", "1", "--color=always")
	if code != 0 {
		t.Fatalf("code %d\n%s", code, out)
	}
	head, _, _ := strings.Cut(out, "\x1b")
	if !strings.Contains(head, "mode cyclo\n") || !strings.Contains(head, "p1 ") {
		t.Fatalf("header colored\n%s", out)
	}
	straight := intensityOf(t, out, "straight.go")
	branch := intensityOf(t, out, "branch.go")
	if straight != minIntensity || branch <= straight {
		t.Fatalf("straight %d branch %d\n%s", straight, branch, out)
	}
	if intensity(1, 9, cycloFloor) != minIntensity || intensity(9, 9, cycloFloor) != maxIntensity || intensity(1, 1, cycloFloor) != neutralIntensity {
		t.Fatal("cyclo floor")
	}
}

func TestModeFlag(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", "package p\n\nfunc B() int {\n\treturn 1\n}\n")
	if _, code := run(dir, "-m", "nope"); code != 2 {
		t.Fatalf("bad mode %d", code)
	}
	if out, code := run(dir, "-m"); code != 2 || !strings.Contains(out, "loc or cyclo") {
		t.Fatalf("bare -m %d\n%s", code, out)
	}
	for _, args := range [][]string{
		{"-mcyclo"},
		{"--mode=cyclo"},
		{"--mode", "cyclo"},
	} {
		out, code := run(dir, args...)
		if code != 0 || !strings.HasPrefix(out, "mode cyclo\n") {
			t.Fatalf("%v code %d\n%s", args, code, out)
		}
	}
}

func TestCycloOmitsUnscoredFiles(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "notes.txt", "one\ntwo\n")
	out, code := run(dir, "-m", "cyclo")
	if code != 0 || !strings.Contains(out, "No modules.") || strings.Contains(out, "notes.txt") {
		t.Fatalf("%d\n%s", code, out)
	}
}

func TestJSONKeepsTheChosenCutoff(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", strings.Repeat("x\n", 10))
	write(t, dir, "b/c.go", strings.Repeat("x\n", 100))
	write(t, dir, "b/d.go", strings.Repeat("x\n", 10))
	out, code := run(dir, "-p", "50", "--json")
	if code != 0 || strings.Contains(out, "\x1b") || strings.Contains(out, "p90") || strings.Contains(out, "No modules.") {
		t.Fatalf("code %d\n%s", code, out)
	}
	var rep jsonReport
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Mode != modeLoc || rep.Modules != 5 || rep.P != 50 || rep.Cutoff != 100 || rep.Note != "" || rep.Tree == nil || rep.Tree.Path != "." {
		t.Fatalf("%+v", rep)
	}
	if len(rep.Tree.Children) != 1 || rep.Tree.Children[0].Path != "b" || rep.Tree.Children[0].Size != 110 {
		t.Fatalf("%+v", rep.Tree)
	}
	kids := rep.Tree.Children[0].Children
	if len(kids) != 1 || kids[0].Path != "b/c.go" || kids[0].Size != 100 {
		t.Fatalf("%+v", kids)
	}
	empty, code := run(t.TempDir(), "--json")
	if code != 0 || strings.Contains(empty, "No modules.") {
		t.Fatalf("empty %d\n%s", code, empty)
	}
	var none jsonReport
	if err := json.Unmarshal([]byte(empty), &none); err != nil {
		t.Fatal(err)
	}
	if none.Modules != 0 || none.P != 95 || none.Tree != nil {
		t.Fatalf("%+v", none)
	}
}

const goBranch = "package p\n\nfunc A(x int) int {\n\tif x > 0 {\n\t\treturn 1\n\t}\n\treturn 0\n}\n\nfunc B() int {\n\treturn 1\n}\n"

const pyBranch = "def a(x):\n    if x > 0:\n        return 1\n    return 0\n\ndef b():\n    return 1\n"

func intensityOf(t *testing.T, out, label string) int {
	t.Helper()
	at := strings.Index(out, label)
	if at < 0 {
		t.Fatalf("missing %q\n%s", label, out)
	}
	mark := strings.LastIndex(out[:at], "\x1b[38;2;")
	if mark < 0 {
		t.Fatalf("no color before %q\n%s", label, out)
	}
	rest := out[mark+len("\x1b[38;2;"):]
	n, err := strconv.Atoi(rest[:strings.IndexByte(rest, ';')])
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func child(n *node, name string) *node {
	return n.by[name]
}

func render(n *node, p int) string {
	var b bytes.Buffer
	writeReport(&b, n, p, modeLoc, false)
	return b.String()
}

func write(t *testing.T, dir, rel, body string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %s", args, out)
	}
}

func run(dir string, args ...string) (string, int) {
	var out, errb bytes.Buffer
	code := runAt(args, dir, &out, &errb)
	return out.String() + errb.String(), code
}
