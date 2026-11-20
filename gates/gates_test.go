package gates

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func call(args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return out.String(), errb.String(), code
}

func TestTypesInOrder(t *testing.T) {
	for _, args := range [][]string{nil, {"list"}} {
		out, _, code := call(args...)
		if code != 0 || out != "code\nessays\n" {
			t.Fatalf("%q code %d\n%s", args, code, out)
		}
	}
}

func TestListCode(t *testing.T) {
	out, _, code := call("list", "code")
	want := sequential + "\nSection 1  Architecture guardrails\n"
	if code != 0 || out != want {
		t.Fatalf("code %d\n%q", code, out)
	}
}

func TestEssaysHaveNoGates(t *testing.T) {
	for _, cmd := range []string{"list", "show"} {
		out, _, code := call(cmd, "essays")
		if code != 0 || out != sequential+"\nno gates\n" {
			t.Fatalf("%s code %d\n%q", cmd, code, out)
		}
	}
}

func TestShowOneGate(t *testing.T) {
	out, _, code := call("show", "code", "1")
	if code != 0 || !strings.HasPrefix(out, "Section 1\nArchitecture guardrails\n\n# Architecture guardrails\n") {
		t.Fatalf("code %d\n%s", code, out)
	}
	if !strings.Contains(out, "stable, immutable identifier") {
		t.Fatalf("statute sentence missing\n%s", out)
	}
	if strings.Contains(out, sequential) {
		t.Fatalf("one gate repeats the sequential line\n%s", out)
	}
}

func TestShowType(t *testing.T) {
	out, _, code := call("show", "code")
	if code != 0 || !strings.HasPrefix(out, sequential+"\n\nSection 1\nArchitecture guardrails\n\n") {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestEmbeddedBodyMatchesFile(t *testing.T) {
	raw, err := os.ReadFile("statute/code/1.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != codeArchitecture || !strings.HasSuffix(codeArchitecture, ".\n") {
		t.Fatal("embedded body differs from statute/code/1.md")
	}
	out, _, _ := call("show", "code", "1")
	if out != "Section 1\nArchitecture guardrails\n\n"+string(raw) {
		t.Fatal("show changed the body")
	}
}

func TestUnknownType(t *testing.T) {
	for _, args := range [][]string{{"show", "poems"}, {"list", "poems"}} {
		out, errOut, code := call(args...)
		if code != 2 || out != "" || !strings.Contains(errOut, "unknown type") {
			t.Fatalf("%q code %d\n%s%s", args, code, out, errOut)
		}
	}
}

func TestUnknownSection(t *testing.T) {
	for _, section := range []string{"2", "1.0", "01", "1.1"} {
		out, errOut, code := call("show", "code", section)
		if code != 2 || out != "" || !strings.Contains(errOut, "unknown section") {
			t.Fatalf("%s code %d\n%s%s", section, code, out, errOut)
		}
	}
}

func TestNoWayToRewrite(t *testing.T) {
	for _, cmd := range []string{"edit", "add", "remove", "nope"} {
		out, errOut, code := call(cmd, "code", "1")
		if code != 2 || out != "" {
			t.Fatalf("%s code %d\n%s", cmd, code, out)
		}
		if !strings.Contains(errOut, "changed and reinstalled") {
			t.Fatalf("%s does not say how the text changes\n%s", cmd, errOut)
		}
	}
}

func TestExtraWordsRejected(t *testing.T) {
	for _, args := range [][]string{{"list", "code", "1"}, {"show"}, {"show", "code", "1", "2"}, {"help", "x"}} {
		if _, _, code := call(args...); code != 2 {
			t.Fatalf("%q code %d", args, code)
		}
	}
}

func TestHelpAndVersion(t *testing.T) {
	for _, word := range []string{"help", "--help", "-h"} {
		out, _, code := call(word)
		if code != 0 || out != Help {
			t.Fatalf("%s code %d", word, code)
		}
	}
	for _, needle := range []string{"code, essays", "c3po gates list", "c3po gates show", sequential,
		"changed and reinstalled", "The agent judges", "Section <number>  <title>  pass", "not judged"} {
		if !strings.Contains(Help, needle) {
			t.Fatalf("help missing %q", needle)
		}
	}
	out, _, code := call("version")
	if code != 0 || !strings.HasPrefix(out, "c3po ") {
		t.Fatalf("version code %d %q", code, out)
	}
}

func TestSectionOrderIsNumeric(t *testing.T) {
	gs := []gate{{section: "10"}, {section: "3.5.1"}, {section: "2.3"}, {section: "3"}, {section: "1"}, {section: "2"}}
	sortGates(gs)
	var got []string
	for _, g := range gs {
		got = append(got, g.section)
	}
	if strings.Join(got, " ") != "1 2 2.3 3 3.5.1 10" {
		t.Fatalf("order %v", got)
	}
}

func TestParseSection(t *testing.T) {
	for _, s := range []string{"1", "2.3", "3.5.1", "10"} {
		if _, ok := parseSection(s); !ok {
			t.Fatalf("%q refused", s)
		}
	}
	for _, s := range []string{"", "0", "1.0", "01", "1..2", "1.", "+1", "-1", "a"} {
		if _, ok := parseSection(s); ok {
			t.Fatalf("%q accepted", s)
		}
	}
}

func TestStatuteShape(t *testing.T) {
	if len(statute) != 2 || statute[0].name != "code" || statute[1].name != "essays" {
		t.Fatalf("types %v", statute)
	}
	if len(statute[0].gates) != 1 || len(statute[1].gates) != 0 {
		t.Fatal("only code section 1 is published")
	}
}
