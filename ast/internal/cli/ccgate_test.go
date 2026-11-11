package cli

import (
	"bytes"
	"encoding/json"
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

const symbolOld, symbolNew = "\treturn 0\n", "\tif x == 99 {\n\t\treturn 99\n\t}\n\treturn 0\n"

func TestPatchCCSymbolCrossingIsRefused(t *testing.T) {
	w := newCCWS(t)
	old := "package p\n\n" + ccFunc("F", 29)
	h := w.write("pkg/f.go", old)
	code, out, errS := w.run("patch", "pkg/f.go", "--expect-hash", h, "--symbol", "F", "--old", symbolOld, "--new", symbolNew)
	if code != ExitFail || out != "refused  pkg/f.go:F  cc 29 -> 30 crosses 30\n" || strings.Contains(errS, "Error") {
		t.Fatalf("code %d out=%q err=%q", code, out, errS)
	}
	if w.read("pkg/f.go") != old {
		t.Fatal("refused symbol patch was written")
	}
}

var symbolOK = regexp.MustCompile(`^ok  [0-9a-f]{64}  pkg/f\.go  formatter \S+$`)

func TestPatchCCSymbolRiseIsWrittenWithoutCCLine(t *testing.T) {
	for _, cc := range []int{5, 30} {
		w := newCCWS(t)
		h := w.write("pkg/f.go", "package p\n\n"+ccFunc("F", cc))
		code, out, errS := w.run("patch", "pkg/f.go", "--expect-hash", h, "--symbol", "F", "--old", symbolOld, "--new", symbolNew)
		if code != 0 {
			t.Fatalf("cc %d: code %d out=%q err=%q", cc, code, out, errS)
		}
		if lines := outLines(out); len(lines) != 1 || !symbolOK.MatchString(lines[0]) {
			t.Fatalf("cc %d: out %q", cc, out)
		}
		if !strings.Contains(w.read("pkg/f.go"), "return 99") {
			t.Fatalf("cc %d: file not written", cc)
		}
	}
}

// fromFile writes content outside the workspace and returns its path.
func fromFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "src")
	mustWrite(t, p, content)
	return p
}

func TestCreateCCNewHighIsRefused(t *testing.T) {
	w := newCCWS(t)
	src := fromFile(t, "package p\n\n"+ccFunc("A", 2)+"\n"+ccFunc("B", 31))
	code, out, errS := w.run("create", "pkg/n.go", "--expect-absent", "--parents", "--from-file", src)
	if code != ExitFail || out != "refused  pkg/n.go:B  cc new -> 31 crosses 30\n" || strings.Contains(errS, "Error") {
		t.Fatalf("code %d out=%q err=%q", code, out, errS)
	}
	if _, err := os.Stat(filepath.Join(w.ws, "pkg")); !os.IsNotExist(err) {
		t.Fatalf("refused create made pkg: %v", err)
	}
}

func TestCreateCCWritesWithoutCCLine(t *testing.T) {
	cases := map[string]string{
		"pkg/f.go":  "package p\n\n" + ccFunc("A", 29),
		"pkg/u.go":  "package p\n\nfunc F( {\n" + ccFunc("B", 40),
		"pkg/f.py":  "def g(x):\n" + strings.Repeat("    if x:\n        pass\n", 40),
		"pkg/f.txt": strings.Repeat("if x\n", 40),
	}
	for rel, content := range cases {
		w := newCCWS(t)
		code, out, errS := w.run("create", rel, "--expect-absent", "--parents", "--from-file", fromFile(t, content))
		if code != 0 {
			t.Fatalf("%s: code %d out=%q err=%q", rel, code, out, errS)
		}
		if lines := outLines(out); len(lines) != 1 || !strings.HasPrefix(lines[0], "ok  ") {
			t.Fatalf("%s: out %q", rel, out)
		}
		if w.read(rel) != content {
			t.Fatalf("%s not written", rel)
		}
	}
}

// manifest is an apply manifest of changes, written outside the workspace.
func manifest(t *testing.T, changes ...map[string]string) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{"files": changes})
	if err != nil {
		t.Fatal(err)
	}
	return fromFile(t, string(b))
}

func TestApplyCCRefusesWholeTransaction(t *testing.T) {
	w := newCCWS(t)
	oldA := "package p\n\n" + ccFunc("F", 29)
	hA := w.write("pkg/a.go", oldA)
	hT := w.write("pkg/c.txt", "x\n")
	m := manifest(t,
		map[string]string{"path": "pkg/c.txt", "expect_hash": hT, "old": "x\n", "new": "y\n"},
		map[string]string{"path": "pkg/b.go", "op": "create", "new": "package p\n\n" + ccFunc("B", 31)},
		map[string]string{"path": "pkg/a.go", "expect_hash": hA, "old": oldA, "new": "package p\n\n" + ccFunc("F", 30)},
	)
	want := "refused  pkg/a.go:F  cc 29 -> 30 crosses 30\nrefused  pkg/b.go:B  cc new -> 31 crosses 30\n"
	for _, args := range [][]string{{"apply", "--from-file", m, "--dry-run"}, {"apply", "--from-file", m}} {
		code, out, errS := w.run(args...)
		if code != ExitFail || out != want || strings.Contains(errS, "Error") {
			t.Fatalf("%v: code %d out=%q err=%q", args, code, out, errS)
		}
	}
	if w.read("pkg/a.go") != oldA || w.read("pkg/c.txt") != "x\n" {
		t.Fatal("refused apply wrote a file")
	}
	if _, err := os.Stat(filepath.Join(w.ws, "pkg/b.go")); !os.IsNotExist(err) {
		t.Fatal("refused apply created b.go")
	}
}

func TestApplyCCWritesWithoutCCLine(t *testing.T) {
	w := newCCWS(t)
	oldHigh := "package p\n\n" + ccFunc("F", 30)
	hHigh := w.write("pkg/high.go", oldHigh)
	g := "\n" + ccFunc("G", 40)
	oldGone := "package p\n\n" + ccFunc("A", 2) + g
	hGone := w.write("pkg/gone.go", oldGone)
	hBad := w.write("pkg/bad.go", "package p\n\nfunc F( {\n")
	hPy := w.write("pkg/f.py", "x = 1\n")
	pyNew := "def g(x):\n" + strings.Repeat("    if x:\n        pass\n", 40)
	m := manifest(t,
		map[string]string{"path": "pkg/high.go", "expect_hash": hHigh, "old": oldHigh, "new": "package p\n\n" + ccFunc("F", 33)},
		map[string]string{"path": "pkg/gone.go", "expect_hash": hGone, "old": g, "new": ""},
		map[string]string{"path": "pkg/bad.go", "expect_hash": hBad, "old": "package p\n", "new": "package p\n\n" + ccFunc("H", 40)},
		map[string]string{"path": "pkg/f.py", "expect_hash": hPy, "old": "x = 1\n", "new": pyNew},
	)
	code, out, errS := w.run("apply", "--from-file", m)
	if code != 0 {
		t.Fatalf("code %d out=%q err=%q", code, out, errS)
	}
	lines := outLines(out)
	if len(lines) != 4 || strings.Contains(out, "cc  ") || strings.Contains(out, "refused") {
		t.Fatalf("out %q", out)
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "ok  ") {
			t.Fatalf("out %q", out)
		}
	}
	if w.read("pkg/high.go") != "package p\n\n"+ccFunc("F", 33) || strings.Contains(w.read("pkg/gone.go"), "func G") || w.read("pkg/f.py") != pyNew {
		t.Fatal("apply did not write")
	}
	if !strings.Contains(w.read("pkg/bad.go"), "func H") {
		t.Fatal("unparsed Go not written")
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
