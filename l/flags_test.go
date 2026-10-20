package l

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runL(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	return stdout.String(), stderr.String(), code
}

func TestHelpPrintsFullHelpToStdout(t *testing.T) {
	for _, arg := range []string{"help", "-h", "--help"} {
		out, errOut, code := runL(t, arg)
		if code != 0 || errOut != "" {
			t.Fatalf("%s: code=%d stderr=%s", arg, code, errOut)
		}
		if out != Help {
			t.Fatalf("%s: expected Help, got %q", arg, out)
		}
	}
	for _, want := range []string{"c3po tui l [path] [flags]", "--color <mode>", "--no-color", "--animate <mode>", "--no-ignore", "--ignore <pattern>", "-d, --depth <n|*>", "200L", "100KB"} {
		if !strings.Contains(Help, want) {
			t.Fatalf("Help missing %q", want)
		}
	}
}

func TestHelpWinsOverAPathAnywhere(t *testing.T) {
	root := t.TempDir()
	out, _, code := runL(t, root, "help")
	if code != 0 || out != Help {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestDirectoryNamedHelpAfterDoubleDash(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "help", "a.go"), "x\n")
	out, errOut, code := runL(t, "--no-color", "--", filepath.Join(root, "help"))
	if code != 0 || !strings.Contains(out, "a.go 2B 1L") {
		t.Fatalf("code=%d stderr=%s out=%s", code, errOut, out)
	}
}

func TestVersionPrintsC3poVersion(t *testing.T) {
	for _, arg := range []string{"version", "--version"} {
		out, _, code := runL(t, arg)
		if code != 0 || !strings.HasPrefix(out, "c3po ") {
			t.Fatalf("%s: code=%d out=%q", arg, code, out)
		}
	}
}

func TestUsageErrorsExitTwoWithHelpOnStderr(t *testing.T) {
	cases := map[string][]string{
		"unknown flag: --bogus":                     {"--bogus"},
		"unknown flag: -":                           {"-"},
		"accepts at most one path":                  {"a", "b"},
		"-d requires a depth":                       {"-d"},
		"--level requires a depth":                  {"--level"},
		"--color requires auto, always, or never":   {"--color"},
		"--animate requires auto, always, or never": {"--animate"},
		"--ignore requires a name or glob":          {"--ignore"},
		"--color must be auto, always, or never":    {"--color=rainbow"},
		"depth must be a non-negative integer or *": {"--depth=-1"},
	}
	for want, args := range cases {
		out, errOut, code := runL(t, args...)
		if code != 2 || out != "" {
			t.Fatalf("%v: code=%d stdout=%q", args, code, out)
		}
		if !strings.HasPrefix(errOut, "l: "+want+"\n\n") || !strings.HasSuffix(errOut, Help) {
			t.Fatalf("%v: stderr=%q", args, errOut)
		}
	}
}

func TestMissingPathExitsOne(t *testing.T) {
	out, errOut, code := runL(t, filepath.Join(t.TempDir(), "missing"))
	if code != 1 || out != "" || !strings.HasPrefix(errOut, "l: ") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errOut)
	}
}

func TestColorFlags(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "app.go"), "one\n")
	cases := []struct {
		args  []string
		color bool
	}{
		{[]string{root}, false},
		{[]string{"--color=always", root}, true},
		{[]string{"--color", "always", root}, true},
		{[]string{"--color=auto", root}, false},
		{[]string{"--color=always", "--no-color", root}, false},
		{[]string{"--no-color", "--color=always", root}, true},
		{[]string{"--color", "never", root}, false},
	}
	for _, tc := range cases {
		out, errOut, code := runL(t, tc.args...)
		if code != 0 {
			t.Fatalf("%v: code=%d stderr=%s", tc.args, code, errOut)
		}
		if got := strings.Contains(out, "\x1b["); got != tc.color {
			t.Fatalf("%v: color=%v, want %v: %q", tc.args, got, tc.color, out)
		}
	}
}

func TestColorAutoUsesStdoutWriter(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if isTerminal(f) || isTerminal(&bytes.Buffer{}) {
		t.Fatal("regular file and buffer are not terminals")
	}
	if !shouldColor(f, "always") || shouldColor(f, "never") || shouldColor(f, "auto") {
		t.Fatal("unexpected color decision")
	}
	if shouldAnimate(f, "auto") || !shouldAnimate(f, "always") || shouldAnimate(f, "never") {
		t.Fatal("unexpected animation decision")
	}
}

func TestDepthAndLevelForms(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a", "b", "c", "deep.go"), "x\n")
	for _, args := range [][]string{
		{"--depth=*", root}, {"-d", "*", root}, {"--level", "4", root}, {"--level=*", root},
	} {
		out, errOut, code := runL(t, args...)
		if code != 0 || !strings.Contains(out, "deep.go") {
			t.Fatalf("%v: code=%d stderr=%s out=%s", args, code, errOut, out)
		}
	}
	out, _, _ := runL(t, root)
	if !strings.Contains(out, "└── b\n") || strings.Contains(out, "── c") || strings.Contains(out, "deep.go") {
		t.Fatalf("default depth 2 exceeded:\n%s", out)
	}
	out, _, _ = runL(t, "-d", "0", root)
	if strings.Count(out, "\n") != 1 {
		t.Fatalf("depth 0 should show the root only:\n%s", out)
	}
}

func TestIgnoreFlags(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "keep.go"), "x\n")
	mustWrite(t, filepath.Join(root, "drop.log"), "x\n")
	mustWrite(t, filepath.Join(root, "gen", "x.go"), "x\n")
	mustWrite(t, filepath.Join(root, "vendor", "y.go"), "x\n")
	mustWrite(t, filepath.Join(root, "node_modules", "dep.js"), "x\n")

	out, _, _ := runL(t, "--ignore", "*.log, gen", "--ignore=vendor", root)
	for _, unwanted := range []string{"drop.log", "gen", "vendor", "node_modules"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("did not expect %q:\n%s", unwanted, out)
		}
	}
	if !strings.Contains(out, "keep.go") {
		t.Fatalf("expected keep.go:\n%s", out)
	}
	out, _, _ = runL(t, "--no-ignore", root)
	if !strings.Contains(out, "node_modules") || !strings.Contains(out, "dep.js") {
		t.Fatalf("expected node_modules with --no-ignore:\n%s", out)
	}
}

func TestInternalDirectoriesAreListed(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "internal", "app.go"), "x\n")
	mustWrite(t, filepath.Join(root, "pkg", "internal", "deep.go"), "x\n")
	out, errOut, code := runL(t, "-d", "*", root)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
	for _, want := range []string{"├── internal\n", "│   └── app.go", "    └── internal\n", "deep.go"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q:\n%s", want, out)
		}
	}
	out, _, _ = runL(t, "-f", "-d", "*", root)
	if !strings.Contains(out, "internal") || !strings.Contains(out, "deep.go") {
		t.Fatalf("expected internal with -f:\n%s", out)
	}
}

func TestOutputFormatAndOrder(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "b.go"), "one\ntwo\n")
	mustWrite(t, filepath.Join(root, "A.txt"), "x")
	mustWrite(t, filepath.Join(root, "zdir", "n.bin"), "\x00\x01")
	mustWrite(t, filepath.Join(root, ".hidden"), "x\n")
	out, _, code := runL(t, root)
	want := filepath.Clean(root) + "\n" +
		"├── zdir\n" +
		"│   └── n.bin 2B\n" +
		"├── A.txt 1B 1L\n" +
		"└── b.go 8B 2L\n"
	if code != 0 || out != want {
		t.Fatalf("got\n%q\nwant\n%q", out, want)
	}
}

func TestFilesOnlyWithNoTextPrintsNothing(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "x.bin"), "\x00")
	out, errOut, code := runL(t, "-f", root)
	if code != 0 || out != "" || errOut != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errOut)
	}
}
