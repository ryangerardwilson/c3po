package cyclotree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCallTreeShowsComplexityAndInternal(t *testing.T) {
	dir := writeFlow(t)
	out, code := run(dir, "Start")
	if code != 0 {
		t.Fatalf("code %d\n%s", code, out)
	}
	for _, want := range []string{"Start  1  1  main.go:", "Work  2  0  internal/hidden/h.go:", "  Work  "} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, ">") || strings.Contains(out, "cycle") {
		t.Fatalf("pipe tree\n%s", out)
	}
}

func TestKeysOpenCallsAndStopOnACycle(t *testing.T) {
	s, err := openTree(writeFlow(t), "Start")
	if err != nil {
		t.Fatal(err)
	}
	s.Key('j')
	if !strings.HasPrefix(cursorLine(s), "> ") || !strings.Contains(cursorLine(s), "Work") {
		t.Fatalf("cursor\n%s", s.Text(true))
	}
	s.Key('l')
	if !strings.Contains(s.Text(false), "cycle") {
		t.Fatalf("cycle\n%s", s.Text(false))
	}
	s.Key('l')
	if !strings.Contains(cursorLine(s), "cycle") {
		t.Fatalf("entered\n%s", s.Text(true))
	}
	s.Key('l')
	if !strings.Contains(cursorLine(s), "cycle") {
		t.Fatalf("cycle opened\n%s", s.Text(true))
	}
	s.Key('h')
	if !strings.HasPrefix(cursorLine(s), "> ") || !strings.Contains(cursorLine(s), "Work") {
		t.Fatalf("caller\n%s", s.Text(true))
	}
	s.Key('h')
	if !strings.HasPrefix(cursorLine(s), "> ") || !strings.Contains(cursorLine(s), "Start") {
		t.Fatalf("root\n%s", s.Text(true))
	}
	s.Key('k')
	if !strings.HasPrefix(cursorLine(s), "> ") || !strings.Contains(cursorLine(s), "Start") {
		t.Fatalf("above root\n%s", s.Text(true))
	}
}

func TestAmbiguousSymbol(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", "package a\n\nfunc Dup() {}\n")
	write(t, dir, "b.go", "package b\n\nfunc Dup() {}\n")
	out, code := run(dir, "Dup")
	if code != 1 || !strings.Contains(out, "ambiguous Dup") || !strings.Contains(out, "a.go:") || !strings.Contains(out, "b.go:") {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestNoSymbol(t *testing.T) {
	out, code := run(t.TempDir(), "Missing")
	if code != 1 || !strings.Contains(out, `no symbol "Missing"`) {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestRustRootHasComplexityAndNoCalls(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "line.rs", "fn parse_line(x: i32) -> i32 {\n    if x > 0 {\n        return x;\n    }\n    0\n}\n")
	out, code := run(dir, "parse_line")
	if code != 0 {
		t.Fatalf("code %d\n%s", code, out)
	}
	if !strings.Contains(out, "parse_line  2  0  line.rs:") {
		t.Fatalf("score\n%s", out)
	}
	if strings.Count(out, "\n") != 1 {
		t.Fatalf("calls\n%s", out)
	}
}

func TestPythonCall(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "app.py", "def start():\n    return work()\n\ndef work():\n    if True:\n        return 1\n    return 0\n")
	out, code := run(dir, "start")
	if code != 0 || !strings.Contains(out, "start  1  1  ") || !strings.Contains(out, "work  2  0  ") {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestHelpAndBadArgs(t *testing.T) {
	out, code := run(t.TempDir(), "help")
	if code != 0 || !strings.Contains(out, "j and k") || !strings.Contains(out, "h moves") || !strings.Contains(out, "levels") || !strings.Contains(out, "vim") || !strings.Contains(out, "shortcut list") || !strings.Contains(out, "full path") {
		t.Fatalf("help %d\n%s", code, out)
	}
	out, code = run(t.TempDir())
	if code != 1 || !strings.Contains(out, "no entrypoint") || strings.Contains(out, "Usage:") {
		t.Fatalf("bare %d\n%s", code, out)
	}
	out, code = run(t.TempDir(), "Start", "--nope")
	if code != 2 || !strings.Contains(out, "unknown flag") {
		t.Fatalf("flag %d\n%s", code, out)
	}
	out, code = run(t.TempDir(), "Start", "missing/dir")
	if code != 1 || !strings.Contains(out, "not a directory") {
		t.Fatalf("dir %d\n%s", code, out)
	}
}

func TestDecodeArrowKeys(t *testing.T) {
	if decodeSeq('j', "") != 'j' || decodeSeq(0x1b, "[A") != 'k' || decodeSeq(0x1b, "[D") != 'h' || decodeSeq(0x1b, "") != 0 {
		t.Fatal("keys")
	}
}

func TestCursorKeepsNameColumns(t *testing.T) {
	s, err := openTree(writeFlow(t), "Start")
	if err != nil {
		t.Fatal(err)
	}
	before := nameColumns(s.Text(true))
	s.Key('j')
	after := nameColumns(s.Text(true))
	if before["Start"] < 0 || before["Start"] != after["Start"] || before["Work"] != after["Work"] {
		t.Fatalf("columns %v -> %v\n%s", before, after, s.Text(true))
	}
}

func nameColumns(text string) map[string]int {
	out := map[string]int{}
	for _, line := range strings.Split(text, "\n") {
		for _, name := range []string{"Start", "Work"} {
			if i := strings.Index(line, name+"  "); i >= 0 {
				out[name] = i
			}
		}
	}
	return out
}

func TestCollapseClosesWhatIsOpenUnderTheRow(t *testing.T) {
	s, err := openTree(writeFlow(t), "Start")
	if err != nil {
		t.Fatal(err)
	}
	s.Key('j')
	s.Key('l')
	if !strings.Contains(s.Text(false), "cycle") {
		t.Fatalf("opened\n%s", s.Text(false))
	}
	for _, line := range strings.Split(s.Text(true), "\n") {
		if strings.Contains(line, "cycle") && !strings.Contains(line, "\x1b[2m") {
			t.Fatalf("cycle line %q", line)
		}
	}
	s.Key('c')
	out := s.Text(false)
	if strings.Contains(out, "cycle") || !strings.Contains(out, "Work") {
		t.Fatalf("collapsed\n%s", out)
	}
	s.Key('h')
	s.Key('c')
	out = s.Text(false)
	if strings.Contains(out, "Work") || !strings.Contains(out, "Start") {
		t.Fatalf("root\n%s", out)
	}
}

func TestLeafIsDimBeforeItIsOpened(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nfunc main() {\n\tStart()\n\tLeaf()\n}\n\nfunc Start() {\n\tLeaf()\n}\n\nfunc Leaf() {}\n")
	s, err := openTree(dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	var startLine, leafLine string
	for _, line := range strings.Split(s.Text(true), "\n") {
		if strings.Contains(line, "Start  ") {
			startLine = line
		}
		if strings.Contains(line, "Leaf  ") {
			leafLine = line
		}
	}
	if !strings.Contains(startLine, "\x1b[1m") || strings.Contains(startLine, "\x1b[2m") {
		t.Fatalf("start %q", startLine)
	}
	if !strings.Contains(leafLine, "\x1b[2m") || strings.Contains(leafLine, "\x1b[1m") {
		t.Fatalf("leaf %q", leafLine)
	}
}
func TestExpandableRowsAreBright(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nfunc main() {\n\tStart()\n}\n\nfunc Start() {}\n")
	s, err := openTree(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(s.Text(false), "\x1b") {
		t.Fatalf("pipe\n%s", s.Text(false))
	}
	var mainLine, startLine string
	for _, line := range strings.Split(s.Text(true), "\n") {
		if strings.Contains(line, "main  ") {
			mainLine = line
		}
		if strings.Contains(line, "Start  ") {
			startLine = line
		}
	}
	if !strings.Contains(mainLine, "\x1b[1m") || strings.Contains(mainLine, "\x1b[2m") {
		t.Fatalf("main %q", mainLine)
	}
	if !strings.Contains(startLine, "\x1b[2m") || strings.Contains(startLine, "\x1b[1m") {
		t.Fatalf("start %q", startLine)
	}
}

func TestPaintKeepsTheLastLineOnScreen(t *testing.T) {
	got := screenBytes("a\nb\n")
	if strings.Contains(got, "\x1b[2J") || !strings.Contains(got, "a\x1b[K\nb\x1b[K\x1b[J") {
		t.Fatalf("%q", got)
	}
	if !strings.Contains(got, "\x1b[38;2;220;220;0m") || !strings.Contains(got, "\x1b[0m") {
		t.Fatalf("yellow %q", got)
	}
}
func cursorLine(s *screen) string {
	for _, line := range strings.Split(s.Text(true), "\n") {
		if strings.Contains(line, ">") {
			return line
		}
	}
	return ""
}

func TestNoArgsTreesTheApplication(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nfunc main() {\n\tStart()\n}\n\nfunc Start() {\n\tWork()\n}\n\nfunc Work() int {\n\tif true {\n\t\treturn 1\n\t}\n\treturn 0\n}\n\nfunc Other() {}\n")
	out, code := run(dir)
	if code != 0 {
		t.Fatalf("code %d\n%s", code, out)
	}
	for _, want := range []string{"main  1  2  main.go:", "  Start  1  1  main.go:", "    Work  2  0  main.go:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "Other") {
		t.Fatalf("unreached\n%s", out)
	}
}

func TestRepeatedCallStaysClosed(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nfunc main() {\n\tA()\n\tB()\n}\n\nfunc A() { C() }\n\nfunc B() { C() }\n\nfunc C() { D() }\n\nfunc D() {}\n")
	out, code := run(dir)
	if code != 0 || strings.Count(out, "D  ") != 1 || strings.Count(out, "C  ") != 2 {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestEntrypointCycleStaysClosed(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nfunc main() {\n\tWork()\n}\n\nfunc Work() {\n\tmain()\n}\n")
	out, code := run(dir)
	if code != 0 || !strings.Contains(out, "cycle") || strings.Count(out, "\n") != 3 {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestTwoEntrypointsAreListed(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "cmd/a/main.go", "package main\n\nfunc main() {}\n")
	write(t, dir, "cmd/b/main.go", "package main\n\nfunc main() {}\n")
	out, code := run(dir)
	if code != 1 || !strings.Contains(out, "ambiguous entrypoint") || !strings.Contains(out, "cmd/a/main.go:") || !strings.Contains(out, "cmd/b/main.go:") {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestFuncMainOutsidePackageMainIsNotAnEntrypoint(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "lib.go", "package other\n\nfunc main() {\n\tStart()\n}\n\nfunc Start() {}\n")
	out, code := run(dir)
	if code != 1 || !strings.Contains(out, "no entrypoint") {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestNoArgsPythonMain(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "app.py", "def main():\n    return work()\n\ndef work():\n    if True:\n        return 1\n    return 0\n")
	out, code := run(dir)
	if code != 0 || !strings.Contains(out, "main  1  1  ") || !strings.Contains(out, "  work  2  0  ") {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestNoArgsRustMain(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "src/main.rs", "fn main() {\n    if true {\n        return;\n    }\n}\n")
	out, code := run(dir)
	if code != 0 || !strings.Contains(out, "main  2  0  src/main.rs:") || strings.Count(out, "\n") != 1 {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestDirectoryUsesThatEntrypoint(t *testing.T) {
	parent := t.TempDir()
	app := filepath.Join(parent, "app")
	write(t, parent, "app/main.go", "package main\n\nfunc main() {\n\tStart()\n}\n\nfunc Start() {}\n")
	out, code := run(parent, app)
	if code != 0 || !strings.Contains(out, "main  1  1  main.go:") || !strings.Contains(out, "  Start  1  0  main.go:") {
		t.Fatalf("code %d\n%s", code, out)
	}
}
func writeFlow(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nfunc Start() {\n\tWork()\n}\n")
	write(t, dir, "internal/hidden/h.go", "package hidden\n\nfunc Work() int {\n\tif true {\n\t\treturn Start()\n\t}\n\treturn 0\n}\n")
	return dir
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

func run(dir string, args ...string) (string, int) {
	var out, errb strings.Builder
	code := runAt(args, dir, &out, &errb)
	return out.String() + errb.String(), code
}

func TestNoArgsFollowsImportedCall(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nimport \"example.com/app/work\"\n\nfunc main() {\n\twork.Start()\n}\n")
	write(t, dir, "work/work.go", "package work\n\nfunc Start() {\n\tHidden()\n}\n\nfunc Hidden() int {\n\tif true {\n\t\treturn 1\n\t}\n\treturn 0\n}\n")
	out, code := run(dir)
	if code != 0 {
		t.Fatalf("code %d\n%s", code, out)
	}
	for _, want := range []string{"main  1  2  main.go:", "  Start  1  1  work/work.go:", "    Hidden  2  0  work.go:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q\n%s", want, out)
		}
	}
}

func TestChainDepth(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nfunc main() {\n\tStart()\n}\n\nfunc Start() {\n\tLeaf()\n}\n\nfunc Leaf() {}\n")
	out, code := run(dir)
	if code != 0 {
		t.Fatalf("code %d\n%s", code, out)
	}
	for _, want := range []string{"main  1  2  main.go:", "  Start  1  1  main.go:", "    Leaf  1  0  main.go:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q\n%s", want, out)
		}
	}
}

func TestSourceStampIgnoresNotes(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	write(t, dir, "notes.txt", "a\n")
	first := sourceStamp(dir)
	write(t, dir, "notes.txt", "completely different notes\n")
	if sourceStamp(dir) != first {
		t.Fatal("notes")
	}
	write(t, dir, "main.go", "package main\n\nfunc main() { println(1) }\n")
	if sourceStamp(dir) == first {
		t.Fatal("source")
	}
}

func TestSettleKeepsCollapseAndCursor(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nfunc main() {\n\tStart()\n}\n\nfunc Start() {\n\tLeaf()\n}\n\nfunc Leaf() {}\n")
	s, err := openTree(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	s.Key('j')
	s.Key('c')
	stamp := sourceStamp(dir)
	write(t, dir, "main.go", "package main\n\n// edited\n\nfunc main() {\n\tStart()\n\tExtra()\n}\n\nfunc Start() {\n\tLeaf()\n}\n\nfunc Leaf() {\n\tif true {\n\t\treturn\n\t}\n}\n\nfunc Extra() {}\n")
	held, stamp2, pending := settle(s, dir, "", stamp, "")
	if held != s || stamp2 != stamp || pending == "" {
		t.Fatalf("debounce")
	}
	got, _, pending2 := settle(s, dir, "", stamp, pending)
	if pending2 != "" || got == s {
		t.Fatal("reload")
	}
	out := got.Text(false)
	if strings.Contains(out, "Leaf") || !strings.Contains(out, "Extra") {
		t.Fatalf("tree\n%s", out)
	}
	if !strings.Contains(cursorLine(got), "Start") {
		t.Fatalf("cursor\n%s", got.Text(true))
	}
}

func TestSettleKeepsTheScreenWhenReloadFails(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	s, err := openTree(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	stamp := sourceStamp(dir)
	write(t, dir, "main.go", "package main\n\nfunc other() {}\n")
	_, _, pending := settle(s, dir, "", stamp, "")
	got, _, pending2 := settle(s, dir, "", stamp, pending)
	if got != s || pending2 != "" || !strings.Contains(got.Text(false), "main") {
		t.Fatalf("kept\n%s", got.Text(false))
	}
}

func TestFrameNamesDepth(t *testing.T) {
	s, err := openTree(writeFlow(t), "Start")
	if err != nil {
		t.Fatal(err)
	}
	frame := s.Frame(24)
	if !strings.Contains(frame, "name  cyclo  depth") || strings.Contains(frame, "j/k") || strings.Contains(frame, "enter vim") {
		t.Fatal(frame)
	}
}

func TestEnterOpensVimAtTheRowLine(t *testing.T) {
	dir := writeFlow(t)
	s, err := openTree(dir, "Start")
	if err != nil {
		t.Fatal(err)
	}
	if applyKeys(s, []rune{'\r'}) != actEdit {
		t.Fatal("enter")
	}
	call, ok := vimCall(dir, s)
	if !ok || call.dir != dir || len(call.args) != 2 || call.args[1] != "main.go" {
		t.Fatalf("%+v %v", call, ok)
	}
	if !strings.Contains(s.Text(false), "Start  1  1  main.go:"+strings.TrimPrefix(call.args[0], "+")) {
		t.Fatalf("%s\n%s", call.args, s.Text(false))
	}
	if applyKeys(s, []rune{'j', '\r'}) != actEdit {
		t.Fatal("work")
	}
	call, ok = vimCall(dir, s)
	if !ok || call.args[1] != "internal/hidden/h.go" {
		t.Fatalf("%v", call.args)
	}
	line := strings.TrimPrefix(call.args[0], "+")
	if !strings.Contains(s.Text(false), "Work  2  0  internal/hidden/h.go:"+line) {
		t.Fatalf("%s\n%s", call.args, s.Text(false))
	}
}

func TestEnterOnARowWithoutAFileDoesNothing(t *testing.T) {
	s := &screen{root: &node{name: "solo", loaded: true, open: true}}
	if applyKeys(s, []rune{'\r'}) != actNone {
		t.Fatal("enter")
	}
	if _, ok := vimCall(t.TempDir(), s); ok {
		t.Fatal("file")
	}
}

func TestRunVimStartsVimWithTheLine(t *testing.T) {
	bin := t.TempDir()
	script := filepath.Join(bin, "vim")
	log := filepath.Join(t.TempDir(), "log")
	body := "#!/bin/sh\nprintf '%s\\n' \"$1\" \"$2\" \"$PWD\" > \"" + log + "\"\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	dir := t.TempDir()
	if err := runVim(vimLaunch{dir: dir, args: []string{"+4", "main.go"}}, nil, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(got), "\n"), "\n")
	if len(lines) != 3 || lines[0] != "+4" || lines[1] != "main.go" {
		t.Fatalf("%q", got)
	}
	gotDir, err := filepath.EvalSymlinks(lines[2])
	if err != nil {
		t.Fatal(err)
	}
	wantDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if gotDir != wantDir {
		t.Fatalf("dir %s want %s", gotDir, wantDir)
	}
}

func TestQuestionMarkTogglesShortcuts(t *testing.T) {
	dir := writeFlow(t)
	s, err := openTree(dir, "Start")
	if err != nil {
		t.Fatal(err)
	}
	s.Key('j')
	before := cursorLine(s)
	if strings.Contains(s.Frame(24), "j/k") {
		t.Fatal("shown")
	}
	s.Key('?')
	frame := s.Frame(24)
	for _, want := range []string{"j/k    move", "open in vim", "?      shortcuts", "q      quit"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing %q\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "Work  2") || !strings.Contains(frame, "Start -> Work") {
		t.Fatalf("tree\n%s", frame)
	}
	s.Key('j')
	s.Key('l')
	s.Key('c')
	if applyKeys(s, []rune{'\r'}) != actNone || applyKeys(s, []rune{'q'}) != actQuit {
		t.Fatal("modal keys")
	}
	s.Key('?')
	if strings.Contains(s.Frame(24), "j/k") || cursorLine(s) != before {
		t.Fatalf("closed\n%s", s.Text(true))
	}
	fresh, err := openTree(dir, "Start")
	if err != nil {
		t.Fatal(err)
	}
	s.Key('?')
	if !adoptScreen(s, fresh).shortcuts {
		t.Fatal("kept")
	}
}

func TestBottomLineIsTheFullPath(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nimport \"example.com/app/work\"\n\nfunc main() {\n\twork.Start()\n}\n")
	write(t, dir, "work/work.go", "package work\n\nfunc Start() {\n\tHidden()\n}\n\nfunc Hidden() int {\n\tif true {\n\t\treturn 1\n\t}\n\treturn 0\n}\n")
	s, err := openTree(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	plain := strings.Split(strings.TrimRight(s.Text(false), "\n"), "\n")
	if len(plain) != 3 || !strings.Contains(plain[2], "Hidden") {
		t.Fatalf("pipe\n%s", strings.Join(plain, "\n"))
	}
	if strings.Contains(strings.Join(plain, "\n"), " -> ") {
		t.Fatalf("pipe chain\n%s", strings.Join(plain, "\n"))
	}
	root := lastLine(s.Frame(24))
	s.Key('j')
	if frameAt(s, 22) != "main -> Start" {
		t.Fatalf("start chain %q", frameAt(s, 22))
	}
	s.Key('j')
	rel, line, ok := cursorFile(s)
	want := fmt.Sprintf("%s:%d", rel, line)
	frame := strings.Split(strings.TrimRight(s.Frame(24), "\n"), "\n")
	row := cursorLine(s)
	if !ok || len(frame) != 24 || frame[22] != "main -> Start -> Hidden" || frame[23] != want || want == root {
		t.Fatalf("lines %d want %q root %q ok %v\n%s", len(frame), want, root, ok, strings.Join(frame, "\n"))
	}
	if !strings.Contains(row, " work.go:") || strings.Contains(row, "work/work.go") {
		t.Fatalf("row %s", row)
	}
	if !strings.Contains(screenBytes(s.Frame(24)), want+"\x1b[K\x1b[J") {
		t.Fatal("paint")
	}
	s.Key('?')
	open := strings.Split(strings.TrimRight(s.Frame(24), "\n"), "\n")
	body := strings.Join(open[:len(open)-2], "\n")
	if len(open) != 24 || open[22] != "main -> Start -> Hidden" || open[23] != want || strings.Contains(body, "Hidden") || !strings.Contains(body, "j/k    move") {
		t.Fatalf("modal\n%s", strings.Join(open, "\n"))
	}
}

func frameAt(s *screen, i int) string {
	lines := strings.Split(strings.TrimRight(s.Frame(24), "\n"), "\n")
	if i < 0 || i >= len(lines) {
		return ""
	}
	return lines[i]
}

func lastLine(frame string) string {
	lines := strings.Split(strings.TrimRight(frame, "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}
	return lines[len(lines)-1]
}

func TestNestedPathIsRelativeToTheCaller(t *testing.T) {
	if pathFromParent("work/work.go", "work/work.go") != "work.go" {
		t.Fatal("same file")
	}
	if pathFromParent("work/work.go", "work/extra/extra.go") != "extra/extra.go" {
		t.Fatal("under the caller")
	}
	if pathFromParent("cmd/app/main.go", "internal/trace/trace.go") != "internal/trace/trace.go" {
		t.Fatal("shorter full path")
	}
	if pathFromParent("main.go", "internal/hidden/h.go") != "internal/hidden/h.go" {
		t.Fatal("top file")
	}
}
