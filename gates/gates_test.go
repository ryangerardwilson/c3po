package gates

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/ryangerardwilson/c3po/internal/version"
)

func call(args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return out.String(), errb.String(), code
}

// receiptLine is the receipt line of a published section, computed here
// from the version, the section, and the body rather than by receipt().
func receiptLine(t *testing.T, section string) string {
	t.Helper()
	for _, p := range architecture {
		if p.section == section {
			sum := sha256.Sum256([]byte(version.String() + "\n" + section + "\n" + p.body))
			return fmt.Sprintf("receipt %x\n", sum[:4])
		}
	}
	t.Fatalf("no section %s", section)
	return ""
}

const mention = "Mention it in the summary of the change.\n"

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
	if code != 0 || !strings.HasPrefix(out, "§1  Data model\n§1.1  Domain truth\n") {
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
	if strings.Contains(out, judging) || strings.Contains(out, "\n\n") {
		t.Fatalf("the provision list carries a banner or blank line\n%s", out)
	}
}

func TestTopSections(t *testing.T) {
	want := map[string][2]string{
		"§1":  {"Data model", "Everything else inherits its shape."},
		"§2":  {"State ownership", "Each fact has exactly one owner and one write path."},
		"§3":  {"Boundaries", "The core domain is independent of UI, database, and frameworks."},
		"§4":  {"How to apply", "Before writing code, read the schema, the module layout, and where the affected state is written."},
		"§5":  {"Severity", "Hard rules are MUST. Defaults are SHOULD."},
		"§6":  {"Checklist", "A no on a hard rule blocks the change."},
		"§7":  {"Reporting", "Report each finding as the rule, where, why it matters, and the fix."},
		"§8":  {"Exceptions", "A rule may be broken deliberately, never silently."},
		"§9":  {"Components and UX", "Independent of language, framework, and platform. Use the project's existing component library and conventions. UI-1 through UI-25 are §9.1 through §9.25."},
		"§10": {"Tests", "Independent of language and test framework. A test should be readable on its own and fail only when behaviour is wrong. TS-1 through TS-20 are §10.1 through §10.20."},
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
		parent, last, _ := strings.Cut(strings.TrimPrefix(p.section, "§"), ".")
		if last == "" {
			continue
		}
		next[parent]++
		if want := "§" + parent + "." + strconv.Itoa(next[parent]); p.section != want {
			t.Fatalf("%s where %s belongs", p.section, want)
		}
	}
}

func TestSectionPrintsTheBodyAndReceipt(t *testing.T) {
	for _, arg := range []string{"§8.3", "8.3", "S.8.3"} {
		if out, _, code := call("code", "architecture", arg); code != 0 || out != mention+receiptLine(t, "§8.3") {
			t.Fatalf("%s code %d\n%q", arg, code, out)
		}
	}
	out, _, code := call("code", "architecture", "§5.1")
	if code != 0 || out != "Hard rules marked MUST: do not write or approve code that violates them.\n"+receiptLine(t, "§5.1") {
		t.Fatalf("code %d\n%q", code, out)
	}
}

func TestReceiptHashesVersionSectionAndBody(t *testing.T) {
	p := provision{"§8.3", "Mention it in the summary", strings.TrimSuffix(mention, "\n")}
	sum := sha256.Sum256([]byte(version.String() + "\n§8.3\n" + p.body))
	if got := receipt(p); got != fmt.Sprintf("%x", sum[:4]) || len(got) != 8 {
		t.Fatalf("receipt %q", got)
	}
	t.Logf("§8 %s", receiptLine(t, "§8"))
	seen := map[string]string{}
	for _, p := range architecture {
		r := receipt(p)
		if other, dup := seen[r]; dup {
			t.Fatalf("%s and %s share receipt %s", p.section, other, r)
		}
		seen[r] = p.section
	}
	if receipt(provision{"§8.4", "", p.body}) == receipt(p) || receipt(provision{"§8.3", "", p.body + "."}) == receipt(p) {
		t.Fatal("the receipt ignores the section or the body")
	}
	if out, _, _ := call("code", "architecture"); strings.Contains(out, "receipt") {
		t.Fatalf("a title line carries a receipt\n%s", out)
	}
}

func exceptions(t *testing.T) string {
	return "A rule may be broken deliberately, never silently.\n" + receiptLine(t, "§8") + "\n" +
		"§8.1  Comment at the site\n§8.2  Contain the violation\n" +
		"§8.3  Mention it in the summary\n§8.4  Legitimate exceptions\n" +
		"4 bodies not shown: c3po gates code architecture 8.{1..4}\n"
}

func TestSectionListsItsSubsections(t *testing.T) {
	for _, arg := range []string{"8", "§8", "S.8"} {
		if out, _, code := call("code", "architecture", arg); code != 0 || out != exceptions(t) {
			t.Fatalf("%s code %d\n%q", arg, code, out)
		}
	}
	out, _, code := call("code", "architecture", "1")
	if code != 0 || !strings.HasPrefix(out, "Everything else inherits its shape.\n"+receiptLine(t, "§1")+"\n§1.1  Domain truth\n") ||
		!strings.HasSuffix(out, "\n§1.19  Red flag: payload unlike the model\n19 bodies not shown: c3po gates code architecture 1.{1..19}\n") ||
		strings.Contains(out, "Goal:") {
		t.Fatalf("code %d\n%q", code, out)
	}
}

func TestShownSubsectionsAreNotCounted(t *testing.T) {
	out, _, code := call("code", "architecture", "8", "8.1", "8.2", "8.3", "8.4")
	if code != 0 || strings.Contains(out, "not shown") || !strings.Contains(out, "§8.4  Legitimate exceptions\n\n§8.1\n") {
		t.Fatalf("code %d\n%s", code, out)
	}
	for _, tc := range []struct{ args, want string }{
		{"8 8.3", "3 bodies not shown: c3po gates code architecture 8.1 8.2 8.4\n"},
		{"8 8.4 8.3", "2 bodies not shown: c3po gates code architecture 8.{1..2}\n"},
		{"8 8.1 8.2 8.3", "1 body not shown: c3po gates code architecture 8.4\n"},
		{"8.2 8", "3 bodies not shown: c3po gates code architecture 8.1 8.3 8.4\n"},
	} {
		out, _, code := call(append([]string{"code", "architecture"}, strings.Fields(tc.args)...)...)
		if code != 0 || strings.Count(out, "not shown") != 1 || !strings.Contains(out, tc.want) {
			t.Fatalf("%s code %d\n%s", tc.args, code, out)
		}
	}
}

func TestAllPrintsEveryProvision(t *testing.T) {
	out, errOut, code := call("code", "architecture", "all")
	if code != 0 || errOut != "" || strings.Contains(out, "not shown") || strings.Contains(out, "  ") {
		t.Fatalf("code %d\n%s%s", code, out, errOut)
	}
	blocks := strings.Split(strings.TrimSuffix(out, "\n"), "\n\n")
	if len(blocks) != 140 || len(architecture) != 140 {
		t.Fatalf("%d blocks for %d provisions", len(blocks), len(architecture))
	}
	for i, p := range architecture {
		if want := p.section + "\n" + p.body + "\n" + strings.TrimSuffix(receiptLine(t, p.section), "\n"); blocks[i] != want {
			t.Fatalf("block %d is %q, want %q", i, blocks[i], want)
		}
	}
	if !strings.Contains(out, "\n§8.3\n"+mention+receiptLine(t, "§8.3")) {
		t.Fatal("all lacks the §8.3 leaf body")
	}
	for _, args := range [][]string{{"all", "8"}, {"8", "all"}, {"§all"}} {
		if _, errOut, code := call(append([]string{"code", "architecture"}, args...)...); code != 2 || !strings.Contains(errOut, "unknown section") {
			t.Fatalf("%q code %d %s", args, code, errOut)
		}
	}
}

func TestOnlyTheNextLevelIsListed(t *testing.T) {
	ps := []provision{{"§8", "Eight", "eight"}, {"§8.1", "One", "one"}, {"§8.1.2.3", "Deep", "deep"}, {"§8.2", "Two", "two"}}
	for i := 1; i <= 6; i++ {
		n := strconv.Itoa(i)
		ps = append(ps, provision{"§8.1." + n, "Leaf " + n, "leaf " + n})
	}
	sortProvisions(ps)
	s := statute{name: "deep", provisions: ps}
	k := kind{name: "code"}
	var out bytes.Buffer
	printBody(&out, k, s, ps[0], nil)
	if out.String() != "eight\nreceipt "+receipt(ps[0])+"\n\n§8.1  One\n§8.2  Two\n2 bodies not shown: c3po gates code deep 8.{1..2}\n" {
		t.Fatalf("§8\n%q", out.String())
	}
	out.Reset()
	printBody(&out, k, s, ps[1], map[string]bool{"§8.1.2": true})
	want := "one\nreceipt " + receipt(ps[1]) + "\n\n"
	for i := 1; i <= 6; i++ {
		want += "§8.1." + strconv.Itoa(i) + "  Leaf " + strconv.Itoa(i) + "\n"
	}
	want += "5 bodies not shown: c3po gates code deep 8.1.1 8.1.3 8.1.4 8.1.5 8.1.6\n"
	if out.String() != want {
		t.Fatalf("§8.1\n%q", out.String())
	}
}

func TestThreeFormsNameOneSection(t *testing.T) {
	for _, forms := range [][]string{{"§6", "6", "S.6"}, {"§8.3", "8.3", "S.8.3"}, {"§6.1", "6.1", "S.6.1"}} {
		want, _, code := call("code", "architecture", forms[0])
		if code != 0 || want == "" || strings.HasPrefix(want, "§") {
			t.Fatalf("%s code %d\n%q", forms[0], code, want)
		}
		for _, form := range forms[1:] {
			if out, _, code := call("code", "architecture", form); code != 0 || out != want {
				t.Fatalf("%s code %d\n%q, want %q", form, code, out, want)
			}
		}
	}
	for arg, want := range map[string]string{"§6": "§6", "6": "§6", "S.6": "§6", "8.3": "§8.3", "S.6.1": "§6.1"} {
		if got, ok := resolveSection(arg); !ok || got != want {
			t.Fatalf("%q resolved to %q %v", arg, got, ok)
		}
	}
}

func TestSeveralSectionsInTheOrderWritten(t *testing.T) {
	six, _, _ := call("code", "architecture", "6")
	seven, _, _ := call("code", "architecture", "7")
	if !strings.Contains(six, "\n\n§6.1  ") || !strings.HasSuffix(six, "\n§6.13  Boundaries: business rule in the core\n13 bodies not shown: c3po gates code architecture 6.{1..13}\n") ||
		!strings.HasSuffix(seven, "\n§7.5  Finding order\n5 bodies not shown: c3po gates code architecture 7.{1..5}\n") {
		t.Fatalf("§6 and §7 do not list their subsections\n%q\n%q", six, seven)
	}
	want := "§6\n" + six + "\n§7\n" + seven + "\n§8.3\n" + mention + receiptLine(t, "§8.3")
	for _, args := range [][]string{{"6", "7", "8.3"}, {"§6", "S.7", "8.3"}} {
		out, errOut, code := call(append([]string{"code", "architecture"}, args...)...)
		if code != 0 || out != want || errOut != "" {
			t.Fatalf("%q code %d\n%q\n%s", args, code, out, errOut)
		}
	}
	out, _, code := call("code", "architecture", "8.3", "8")
	want = "§8.3\n" + mention + receiptLine(t, "§8.3") + "\n§8\n" +
		strings.Replace(exceptions(t), "4 bodies not shown: c3po gates code architecture 8.{1..4}", "3 bodies not shown: c3po gates code architecture 8.1 8.2 8.4", 1)
	if code != 0 || out != want {
		t.Fatalf("statute order instead of the order written: code %d\n%q", code, out)
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
				if code != 0 || !strings.HasPrefix(out, p.body+"\n") {
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
		{[]string{"code", "architecture", "§11"}, "unknown section"},
		{[]string{"code", "architecture", "§99"}, "unknown section"},
		{[]string{"code", "architecture", "§0"}, "unknown section"},
		{[]string{"code", "architecture", "§1.0"}, "unknown section"},
		{[]string{"code", "architecture", "§01"}, "unknown section"},
		{[]string{"code", "architecture", "§ 1"}, "unknown section"},
		{[]string{"code", "architecture", "§.1"}, "unknown section"},
		{[]string{"code", "architecture", "11"}, "unknown section"},
		{[]string{"code", "architecture", "6.0"}, "unknown section"},
		{[]string{"code", "architecture", "01"}, "unknown section"},
		{[]string{"code", "architecture", "0"}, "unknown section"},
		{[]string{"code", "architecture", "S.0"}, "unknown section"},
		{[]string{"code", "architecture", "S.01"}, "unknown section"},
		{[]string{"code", "architecture", "S.§1"}, "unknown section"},
		{[]string{"code", "architecture", "S§1"}, "unknown section"},
		{[]string{"code", "architecture", "S1"}, "unknown section"},
		{[]string{"code", "architecture", "s.1"}, "unknown section"},
		{[]string{"code", "architecture", ""}, "unknown section"},
		{[]string{"code", "architecture", "C/A/S.1"}, "unknown section"},
		{[]string{"code", "architecture", "6", "11", "7"}, `unknown section "11"`},
		{[]string{"code", "architecture", "§6", "§1", "6.0"}, `unknown section "6.0"`},
		{[]string{"code", "architecture", "§1", "x"}, `unknown section "x"`},
	} {
		out, errOut, code := call(tc.args...)
		if code != 2 || out != "" || !strings.Contains(errOut, tc.want) {
			t.Fatalf("%q code %d\n%s%s", tc.args, code, out, errOut)
		}
	}
}

func TestNoWayToRewrite(t *testing.T) {
	for _, args := range [][]string{{"list"}, {"list", "code"}, {"get", "C/A/S.1"}, {"show", "code"},
		{"edit", "§1"}, {"add"}, {"remove", "code", "architecture", "§1"}} {
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
	for _, args := range [][]string{{"help", "x"}, {"version", "x"}} {
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
		"c3po gates <type> <statute> <section>...\n", "c3po gates code architecture §1.1\n",
		"c3po gates code architecture 6 7 8.3\n", "c3po gates <type> <statute> all\n", "c3po gates code architecture all\n",
		"c3po gates <type> <statute> tally <sheet>\n", "  receipt <8 hex>\n", "  §1.2 <receipt> <pass|fail> <site> [files <path>...]\n",
		"  §8.3 " + receipt(provision{"§8.3", "", strings.TrimSuffix(mention, "\n")}) + " fail store/order.go:41 files store/order.go\n",
		"4 bodies not shown: c3po gates code architecture 8.{1..4}", "all is not a section", "unfixed", "unclaimed",
		"§1.1, 1.1, or S.1.1", "Several sections after\nthe statute",
		"A section with subsections lists those subsections under the body",
		"code, essays", "architecture",
		"300", judging, "Every body is still judged",
		"The agent judges", "c3po only prints the text", "no gates screen", "no way to edit a gate",
		"changed and reinstalled", "stricter than style, framework choice, or\nmicro-optimization"} {
		if !strings.Contains(Help, needle) {
			t.Fatalf("help missing %q", needle)
		}
	}
	for _, gone := range []string{"gates show", "gates list", "gates get", "C/A/", "<id>", "S.1  ", "not judged", "later gates"} {
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
	ps := []provision{{section: "§10"}, {section: "§3.5.1"}, {section: "§2.3"}, {section: "§3"}, {section: "§1"}, {section: "§2"}}
	sortProvisions(ps)
	var got []string
	for _, p := range ps {
		got = append(got, p.section)
	}
	if strings.Join(got, " ") != "§1 §2 §2.3 §3 §3.5.1 §10" {
		t.Fatalf("order %v", got)
	}
}

func TestParseSection(t *testing.T) {
	for _, s := range []string{"§1", "§2.3", "§3.5.1", "§10"} {
		if _, ok := parseSection(s); !ok {
			t.Fatalf("%q refused", s)
		}
	}
	for _, s := range []string{"", "1", "§", "§.", "§.1", "§ 1", "S.1", "s.1", "§0", "§1.0", "§01", "§1..2", "§1.",
		"§+1", "§-1", "§a", "S§1", "§1 "} {
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
