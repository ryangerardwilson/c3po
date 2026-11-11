package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/internal/gocyclo"
)

// pyFunc is a Python function named name whose cyclomatic number is cc.
func pyFunc(name string, cc int) string {
	var b strings.Builder
	b.WriteString("def " + name + "(x):\n")
	for i := 1; i < cc; i++ {
		b.WriteString("    if x == 1:\n        return 1\n")
	}
	b.WriteString("    return 0\n")
	return b.String()
}

// gapWS is a workspace with one Go function at max_cc, one under it, one
// Python function over it, and two high Go functions the walk skips.
func gapWS(t *testing.T) ccWS {
	w := newCCWS(t)
	w.write("a.go", "package p\n\n"+ccFunc("Hi", 30)+"\n"+ccFunc("Lo", 29))
	w.write("b.py", pyFunc("g", 31))
	w.write("notes.txt", strings.Repeat("if x\n", 40))
	w.write("vendor/v.go", "package v\n\n"+ccFunc("V", 50))
	w.write(".gitignore", "gen/\n")
	w.write("gen/g.go", "package g\n\n"+ccFunc("G", 45))
	return w
}

func TestGapListsFunctionsAtOrAboveMaxCC(t *testing.T) {
	w := gapWS(t)
	want := "gap  " + w.ws + "\nmax_cc  30\nover  2\n31  b.py:g  python\n30  a.go:Hi  go\n"
	code, out, errS := w.run("gap")
	if code != 0 || out != want || errS != "" {
		t.Fatalf("code %d out=%q err=%q", code, out, errS)
	}
	code, out, errS = w.run("gap", "--check")
	if code != ExitFail || out != want || errS != "" {
		t.Fatalf("check: code %d out=%q err=%q", code, out, errS)
	}
	scores, ok := gocyclo.File([]byte(w.read("a.go")))
	if !ok || scores["Hi"] != 30 || scores["Lo"] != 29 {
		t.Fatalf("gocyclo %v", scores)
	}
}

func TestGapJSON(t *testing.T) {
	w := gapWS(t)
	code, out, errS := w.run("gap", "--json")
	if code != 0 || errS != "" {
		t.Fatalf("code %d out=%q err=%q", code, out, errS)
	}
	keys := []string{`"root"`, `"max_cc"`, `"over"`, `"functions"`}
	for i := 1; i < len(keys); i++ {
		if strings.Index(out, keys[i-1]) > strings.Index(out, keys[i]) {
			t.Fatalf("key order %q", out)
		}
	}
	var rep gapReport
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatal(err)
	}
	want := []gapFn{
		{Path: "b.py", Name: "g", Language: "python", Line: 1, Cyclomatic: 31},
		{Path: "a.go", Name: "Hi", Language: "go", Line: 3, Cyclomatic: 30},
	}
	if rep.Root != w.ws || rep.MaxCC != 30 || rep.Over != 2 || len(rep.Functions) != 2 || rep.Functions[0] != want[0] || rep.Functions[1] != want[1] {
		t.Fatalf("report %+v", rep)
	}
	code, _, _ = w.run("gap", "--json", "--check")
	if code != ExitFail {
		t.Fatalf("json check code %d", code)
	}
}

func TestGapEmptyDebt(t *testing.T) {
	w := newCCWS(t)
	w.write("a.go", "package p\n\n"+ccFunc("Lo", 29))
	for _, args := range [][]string{{"gap"}, {"gap", "--check"}} {
		code, out, errS := w.run(args...)
		if code != 0 || out != "gap  "+w.ws+"\nmax_cc  30\nover  0\n" || errS != "" {
			t.Fatalf("%v: code %d out=%q err=%q", args, code, out, errS)
		}
	}
	code, out, _ := w.run("gap", "--json")
	var rep gapReport
	if err := json.Unmarshal([]byte(out), &rep); code != 0 || err != nil || rep.Over != 0 || rep.Functions == nil {
		t.Fatalf("code %d out=%q err=%v", code, out, err)
	}
}

func TestGapRootFlag(t *testing.T) {
	w := gapWS(t)
	other := newCCWS(t)
	other.cfg = w.cfg
	code, out, _ := other.run("--root", w.ws, "gap")
	if code != 0 || !strings.HasPrefix(out, "gap  "+w.ws+"\n") || !strings.Contains(out, "over  2\n") {
		t.Fatalf("code %d out=%q", code, out)
	}
}

func TestGapUsage(t *testing.T) {
	w := newCCWS(t)
	for _, args := range [][]string{{"gap", "extra"}, {"gap", "--bogus"}} {
		code, out, errS := w.run(args...)
		if code != ExitUsage || out != "" || !strings.Contains(errS, "Error:") {
			t.Fatalf("%v: code %d out=%q err=%q", args, code, out, errS)
		}
	}
}
