package gates

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func call(args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return out.String(), errb.String(), code
}

func TestTypesInOrder(t *testing.T) {
	out, _, code := call()
	if code != 0 || out != "code\nessays\n" {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestListStatutes(t *testing.T) {
	out, _, code := call("code")
	if code != 0 || out != "architecture\n" {
		t.Fatalf("code %d\n%q", code, out)
	}
	out, _, code = call("essays")
	if code != 0 || out != "no statutes\n" {
		t.Fatalf("essays code %d\n%q", code, out)
	}
}

func TestListProvisions(t *testing.T) {
	out, _, code := call("code", "architecture")
	if code != 0 || !strings.HasPrefix(out, "S.1  Data model\nS.1.1  Domain truth\n") {
		t.Fatalf("code %d\n%s", code, out)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != len(architecture) {
		t.Fatalf("%d lines for %d provisions", len(lines), len(architecture))
	}
	var prev []int
	for _, line := range lines {
		section, heading, ok := strings.Cut(line, "  ")
		nums, parsed := parseSection(section)
		if !ok || heading == "" || !parsed {
			t.Fatalf("bad line %q", line)
		}
		if prev != nil && !sectionLess(prev, nums) {
			t.Fatalf("%q is out of section order", line)
		}
		prev = nums
	}
	if strings.Contains(out, sequential) || strings.Contains(out, "\n\n") {
		t.Fatalf("the provision list carries a banner or blank line\n%s", out)
	}
}

func TestTopSections(t *testing.T) {
	want := map[string][2]string{
		"S.1": {"Data model", "Everything else inherits its shape."},
		"S.2": {"State ownership", "Each fact has exactly one owner and one write path."},
		"S.3": {"Boundaries", "The core domain is independent of UI, database, and frameworks."},
		"S.4": {"How to apply", "Before writing code, read the schema, the module layout, and where the affected state is written."},
		"S.5": {"Severity", "Hard rules are MUST. Defaults are SHOULD."},
		"S.6": {"Checklist", "A no on a hard rule blocks the change."},
		"S.7": {"Reporting", "Report each finding as the rule, where, why it matters, and the fix."},
		"S.8": {"Exceptions", "A rule may be broken deliberately, never silently."},
	}
	var top int
	for _, p := range architecture {
		if nums, _ := parseSection(p.section); len(nums) == 1 {
			top++
			if w := want[p.section]; p.heading != w[0] || p.body != w[1] {
				t.Fatalf("%s is %q %q", p.section, p.heading, p.body)
			}
		}
	}
	if top != len(want) {
		t.Fatalf("%d top sections", top)
	}
}

func TestSubsectionsHaveNoGaps(t *testing.T) {
	next := map[string]int{}
	for _, p := range architecture {
		parent, last, _ := strings.Cut(p.section[2:], ".")
		if last == "" {
			continue
		}
		next[parent]++
		if want := "S." + parent + "." + strconv.Itoa(next[parent]); p.section != want {
			t.Fatalf("%s where %s belongs", p.section, want)
		}
	}
}

func TestSectionPrintsOnlyTheBody(t *testing.T) {
	out, _, code := call("code", "architecture", "S.1")
	if code != 0 || out != "Everything else inherits its shape.\n" {
		t.Fatalf("code %d\n%q", code, out)
	}
	out, _, code = call("code", "architecture", "S.5.1")
	if code != 0 || out != "Hard rules marked MUST: do not write or approve code that violates them.\n" {
		t.Fatalf("code %d\n%q", code, out)
	}
}

func TestBodiesFitTheLimit(t *testing.T) {
	for _, k := range law {
		for _, s := range k.statutes {
			for _, p := range s.provisions {
				id := k.name + " " + s.name + " " + p.section
				if n := utf8.RuneCountInString(p.body); n > maxBody || n == 0 {
					t.Errorf("%s body is %d code points", id, n)
				}
				if n := utf8.RuneCountInString(p.heading); n > 40 || n == 0 {
					t.Errorf("%s heading is %d code points", id, n)
				}
				if strings.ContainsAny(p.body, "*`\n") || strings.HasPrefix(p.body, "- ") {
					t.Errorf("%s body keeps markdown", id)
				}
				out, _, code := call(k.name, s.name, p.section)
				if code != 0 || out != p.body+"\n" {
					t.Errorf("%s changed the body", id)
				}
			}
		}
	}
}

func TestRulesKept(t *testing.T) {
	for _, phrase := range []string{"stable, immutable identifier", "exactly one write path",
		"Dependencies MUST point inward", "inherits its shape"} {
		var found bool
		for _, p := range architecture {
			found = found || strings.Contains(p.body, phrase)
		}
		if !found {
			t.Errorf("no body says %q", phrase)
		}
	}
}

func TestUnknownNames(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"poems"}, "unknown type"},
		{[]string{"nope", "C/A/S.1"}, "unknown type"},
		{[]string{"Code"}, "unknown type"},
		{[]string{"code", "style"}, "unknown statute"},
		{[]string{"essays", "architecture"}, "unknown statute"},
		{[]string{"code", "architecture", "S.9"}, "unknown section"},
		{[]string{"code", "architecture", "S.99"}, "unknown section"},
		{[]string{"code", "architecture", "S.1.0"}, "unknown section"},
		{[]string{"code", "architecture", "S.01"}, "unknown section"},
		{[]string{"code", "architecture", "1"}, "unknown section"},
		{[]string{"code", "architecture", "s.1"}, "unknown section"},
		{[]string{"code", "architecture", "C/A/S.1"}, "unknown section"},
	} {
		out, errOut, code := call(tc.args...)
		if code != 2 || out != "" || !strings.Contains(errOut, tc.want) {
			t.Fatalf("%q code %d\n%s%s", tc.args, code, out, errOut)
		}
	}
}

func TestNoWayToRewrite(t *testing.T) {
	for _, args := range [][]string{{"list"}, {"list", "code"}, {"get", "C/A/S.1"}, {"show", "code"},
		{"edit", "S.1"}, {"add"}, {"remove", "code", "architecture", "S.1"}} {
		cmd := args[0]
		out, errOut, code := call(args...)
		if code != 2 || out != "" {
			t.Fatalf("%s code %d\n%s", cmd, code, out)
		}
		if !strings.Contains(errOut, "unknown command") || !strings.Contains(errOut, "changed and reinstalled") {
			t.Fatalf("%s does not say how the text changes\n%s", cmd, errOut)
		}
	}
}

func TestExtraWordsRejected(t *testing.T) {
	for _, args := range [][]string{{"code", "architecture", "S.1", "S.2"}, {"code", "architecture", "S.1", "x"},
		{"help", "x"}, {"version", "x"}} {
		out, _, code := call(args...)
		if code != 2 || out != "" {
			t.Fatalf("%q code %d\n%s", args, code, out)
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
	for _, needle := range []string{"  c3po gates\n", "c3po gates <type>\n", "c3po gates <type> <statute>\n",
		"c3po gates <type> <statute> <section>\n", "code, essays", "architecture",
		"300", sequential, "  S.1  pass\n", "  S.1  fail\n", "  S.1.1  fail  not judged\n",
		"The agent judges", "c3po only prints the text", "no gates screen", "no way to edit a gate",
		"changed and reinstalled", "stricter than style, framework choice, or\nmicro-optimization"} {
		if !strings.Contains(Help, needle) {
			t.Fatalf("help missing %q", needle)
		}
	}
	for _, gone := range []string{"gates show", "gates list", "gates get", "C/A/", "<id>"} {
		if strings.Contains(Help, gone) {
			t.Fatalf("help names %q", gone)
		}
	}
	out, _, code := call("version")
	if code != 0 || !strings.HasPrefix(out, "c3po ") {
		t.Fatalf("version code %d %q", code, out)
	}
}

func TestSectionOrderIsNumeric(t *testing.T) {
	ps := []provision{{section: "S.10"}, {section: "S.3.5.1"}, {section: "S.2.3"}, {section: "S.3"}, {section: "S.1"}, {section: "S.2"}}
	sortProvisions(ps)
	var got []string
	for _, p := range ps {
		got = append(got, p.section)
	}
	if strings.Join(got, " ") != "S.1 S.2 S.2.3 S.3 S.3.5.1 S.10" {
		t.Fatalf("order %v", got)
	}
}

func TestParseSection(t *testing.T) {
	for _, s := range []string{"S.1", "S.2.3", "S.3.5.1", "S.10"} {
		if _, ok := parseSection(s); !ok {
			t.Fatalf("%q refused", s)
		}
	}
	for _, s := range []string{"", "1", "S", "S.", "s.1", "S.0", "S.1.0", "S.01", "S.1..2", "S.1.", "S.+1", "S.-1", "S.a"} {
		if _, ok := parseSection(s); ok {
			t.Fatalf("%q accepted", s)
		}
	}
}

func TestLawShape(t *testing.T) {
	if len(law) != 2 || law[0].name != "code" || law[1].name != "essays" {
		t.Fatalf("types %v", law)
	}
	if len(law[0].statutes) != 1 || law[0].statutes[0].name != "architecture" || len(law[1].statutes) != 0 {
		t.Fatal("only code architecture is published")
	}
}
