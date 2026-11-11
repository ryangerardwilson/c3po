package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/files"
)

// ccFunc is a Go function named name whose cyclomatic number is cc.
func ccFunc(name string, cc int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "func %s(x int) int {\n", name)
	for i := 1; i < cc; i++ {
		fmt.Fprintf(&b, "\tif x == %d {\n\t\treturn %d\n\t}\n", i, i)
	}
	b.WriteString("\treturn 0\n}\n")
	return b.String()
}

type ccWS struct {
	t   *testing.T
	ws  string
	cfg string
}

func newCCWS(t *testing.T) ccWS {
	t.Helper()
	ws := t.TempDir()
	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	cfg := config.Default()
	cfg.Enforcement.Mode = "never"
	if err := config.Write(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	return ccWS{t, ws, cfgPath}
}

func (w ccWS) write(rel, content string) string {
	w.t.Helper()
	abs := filepath.Join(w.ws, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		w.t.Fatal(err)
	}
	mustWrite(w.t, abs, content)
	h, err := files.HashFile(abs)
	if err != nil {
		w.t.Fatal(err)
	}
	return h
}

func (w ccWS) read(rel string) string {
	w.t.Helper()
	data, err := os.ReadFile(filepath.Join(w.ws, rel))
	if err != nil {
		w.t.Fatal(err)
	}
	return string(data)
}

func (w ccWS) run(args ...string) (int, string, string) {
	var out, errB bytes.Buffer
	r := NewRunner(&out, &errB)
	r.Cwd = w.ws
	r.ConfigPath = w.cfg
	code := r.Run(args)
	return code, out.String(), errB.String()
}

func outLines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

var okLine = regexp.MustCompile(`^ok  [0-9a-f]{64}  pkg/f\.go$`)

func TestPatchCCRiseUnderMaxIsWritten(t *testing.T) {
	w := newCCWS(t)
	old := "package p\n\n" + ccFunc("F", 5)
	h := w.write("pkg/f.go", old)
	newSrc := "package p\n\n" + ccFunc("F", 6)
	code, out, errS := w.run("patch", "pkg/f.go", "--expect-hash", h, "--old", old, "--new", newSrc)
	if code != 0 {
		t.Fatalf("code %d out=%q err=%q", code, out, errS)
	}
	lines := outLines(out)
	if len(lines) != 2 || !okLine.MatchString(lines[0]) || lines[1] != "cc  pkg/f.go:F  5 -> 6" {
		t.Fatalf("out %q", out)
	}
	if w.read("pkg/f.go") != newSrc {
		t.Fatal("file not written")
	}
}

func TestPatchCCCrossingIsRefused(t *testing.T) {
	w := newCCWS(t)
	old := "package p\n\n" + ccFunc("F", 29)
	h := w.write("pkg/f.go", old)
	newSrc := "package p\n\n" + ccFunc("F", 30)
	code, out, errS := w.run("patch", "pkg/f.go", "--expect-hash", h, "--old", old, "--new", newSrc)
	if code != ExitFail {
		t.Fatalf("code %d out=%q err=%q", code, out, errS)
	}
	if out != "refused  pkg/f.go:F  cc 29 -> 30 crosses 30\n" {
		t.Fatalf("out %q", out)
	}
	if strings.Contains(errS, "Error") {
		t.Fatalf("stderr %q", errS)
	}
	if w.read("pkg/f.go") != old {
		t.Fatal("refused patch was written")
	}
}

func TestPatchCCAlreadyHighMayRise(t *testing.T) {
	w := newCCWS(t)
	old := "package p\n\n" + ccFunc("F", 30)
	h := w.write("pkg/f.go", old)
	newSrc := "package p\n\n" + ccFunc("F", 33)
	code, out, errS := w.run("patch", "pkg/f.go", "--expect-hash", h, "--old", old, "--new", newSrc)
	if code != 0 {
		t.Fatalf("code %d out=%q err=%q", code, out, errS)
	}
	if lines := outLines(out); len(lines) != 2 || lines[1] != "cc  pkg/f.go:F  30 -> 33" {
		t.Fatalf("out %q", out)
	}
	if w.read("pkg/f.go") != newSrc {
		t.Fatal("file not written")
	}
}

func TestPatchCCNewHighFunctionIsRefused(t *testing.T) {
	w := newCCWS(t)
	old := "package p\n\n" + ccFunc("A", 2)
	h := w.write("pkg/f.go", old)
	newSrc := old + "\n" + ccFunc("B", 31) + "\n" + ccFunc("C", 4)
	code, out, errS := w.run("patch", "pkg/f.go", "--expect-hash", h, "--old", old, "--new", newSrc)
	if code != ExitFail || out != "refused  pkg/f.go:B  cc new -> 31 crosses 30\n" || strings.Contains(errS, "Error") {
		t.Fatalf("code %d out=%q err=%q", code, out, errS)
	}
	if w.read("pkg/f.go") != old {
		t.Fatal("refused patch was written")
	}
}

func TestPatchCCRemovedFunctionIsGone(t *testing.T) {
	w := newCCWS(t)
	g := "\n" + ccFunc("G", 3)
	old := "package p\n\n" + ccFunc("F", 2) + g
	h := w.write("pkg/f.go", old)
	code, out, errS := w.run("patch", "pkg/f.go", "--expect-hash", h, "--old", g, "--new", "")
	if code != 0 {
		t.Fatalf("code %d out=%q err=%q", code, out, errS)
	}
	lines := outLines(out)
	if len(lines) != 2 || !strings.HasSuffix(lines[1], "gone") || lines[1] != "cc  pkg/f.go:G  3 -> gone" {
		t.Fatalf("out %q", out)
	}
	if strings.Contains(w.read("pkg/f.go"), "func G") {
		t.Fatal("file not written")
	}
}

func TestPatchCCNonGoHasNoCCLine(t *testing.T) {
	for _, rel := range []string{"pkg/f.py", "pkg/f.txt"} {
		w := newCCWS(t)
		old := strings.Repeat("if x:\n    pass\n", 40)
		h := w.write(rel, "x = 1\n")
		code, out, errS := w.run("patch", rel, "--expect-hash", h, "--old", "x = 1\n", "--new", old)
		if code != 0 {
			t.Fatalf("%s: code %d out=%q err=%q", rel, code, out, errS)
		}
		if lines := outLines(out); len(lines) != 1 || !strings.HasPrefix(lines[0], "ok  ") || strings.Contains(out, "cc  ") {
			t.Fatalf("%s: out %q", rel, out)
		}
		if w.read(rel) != old {
			t.Fatalf("%s not written", rel)
		}
	}
}

func TestPatchCCUnparsedGoIsWritten(t *testing.T) {
	w := newCCWS(t)
	h := w.write("pkg/f.go", "package p\n\nfunc F( {\n")
	code, out, errS := w.run("patch", "pkg/f.go", "--expect-hash", h, "--old", "F(", "--new", "G(")
	if code != 0 {
		t.Fatalf("code %d out=%q err=%q", code, out, errS)
	}
	lines := outLines(out)
	if len(lines) != 2 || !okLine.MatchString(lines[0]) || lines[1] != "cc  pkg/f.go  unparsed" {
		t.Fatalf("out %q", out)
	}
	if !strings.Contains(w.read("pkg/f.go"), "func G(") {
		t.Fatal("file not written")
	}
}

func TestPatchCCDuplicateNameIsUnparsed(t *testing.T) {
	w := newCCWS(t)
	old := "package p\n\nfunc init() {}\n\nfunc init() {}\n"
	h := w.write("pkg/f.go", old)
	code, out, _ := w.run("patch", "pkg/f.go", "--expect-hash", h, "--old", "package p\n", "--new", "package p\n\n"+ccFunc("H", 40))
	if code != 0 {
		t.Fatalf("code %d out=%q", code, out)
	}
	if lines := outLines(out); len(lines) != 2 || lines[1] != "cc  pkg/f.go  unparsed" {
		t.Fatalf("out %q", out)
	}
}

func TestPatchCCHashMismatchHasNoCCLine(t *testing.T) {
	w := newCCWS(t)
	old := "package p\n\n" + ccFunc("F", 5)
	w.write("pkg/f.go", old)
	bad := strings.Repeat("0", 64)
	code, out, errS := w.run("patch", "pkg/f.go", "--expect-hash", bad, "--old", old, "--new", "package p\n\n"+ccFunc("F", 40))
	if code != ExitFail || strings.Contains(out, "cc  ") || strings.Contains(out, "refused") || !strings.Contains(errS, "hash mismatch") {
		t.Fatalf("code %d out=%q err=%q", code, out, errS)
	}
	if w.read("pkg/f.go") != old {
		t.Fatal("file written on mismatch")
	}
}

func TestPatchCCSymbolPatchIsNotGated(t *testing.T) {
	w := newCCWS(t)
	old := "package p\n\n" + ccFunc("F", 29)
	h := w.write("pkg/f.go", old)
	newSrc := "package p\n\n" + ccFunc("F", 30)
	code, out, errS := w.run("patch", "pkg/f.go", "--expect-hash", h, "--symbol", "F",
		"--old", "\treturn 0\n", "--new", "\tif x == 29 {\n\t\treturn 29\n\t}\n\treturn 0\n")
	if code != 0 {
		t.Fatalf("code %d out=%q err=%q", code, out, errS)
	}
	if strings.Contains(out, "refused") || strings.Contains(out, "cc  ") || !strings.Contains(out, "formatter") {
		t.Fatalf("out %q", out)
	}
	if w.read("pkg/f.go") != newSrc {
		t.Fatalf("file %q", w.read("pkg/f.go"))
	}
}

func TestPatchCCHonorsConfiguredMax(t *testing.T) {
	w := newCCWS(t)
	cfg := config.Default()
	cfg.Enforcement.Mode = "never"
	cfg.MaxCC = 6
	if err := config.Write(w.cfg, cfg); err != nil {
		t.Fatal(err)
	}
	old := "package p\n\ntype T struct{}\n\nfunc (t *T) M(x int) int {\n\treturn x\n}\n"
	h := w.write("pkg/f.go", old)
	newSrc := strings.Replace(old, "\treturn x\n", "\tif x > 0 && x < 9 || x == 20 {\n\t\treturn 1\n\t}\n\tswitch x {\n\tcase 1:\n\tcase 2:\n\t}\n\treturn x\n", 1)
	code, out, _ := w.run("patch", "pkg/f.go", "--expect-hash", h, "--old", old, "--new", newSrc)
	if code != ExitFail || out != "refused  pkg/f.go:T.M  cc 1 -> 6 crosses 6\n" {
		t.Fatalf("code %d out=%q", code, out)
	}
}
