package gates

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// maxBody is the longest body, in Unicode code points, a provision may publish.
const maxBody = 300

// provision is one numbered rule of a statute.
type provision struct {
	section string
	heading string
	body    string
}

// statute is a named body of provisions in section order.
type statute struct {
	name       string
	provisions []provision
}

// kind is a broad subject and its statutes.
type kind struct {
	name     string
	statutes []statute
}

// law is the published legislation, in type order. It is compiled into the
// binary; there is no file, directory, or command that changes it.
var law = []kind{
	{name: "code", statutes: []statute{{name: "architecture", provisions: architecture}}},
	{name: "essays"},
}

func init() {
	for _, k := range law {
		for _, s := range k.statutes {
			seen := map[string]bool{}
			for _, p := range s.provisions {
				if _, ok := parseSection(p.section); !ok || seen[p.section] {
					panic(fmt.Sprintf("gates: bad or repeated section %q in %s %s", p.section, k.name, s.name))
				}
				seen[p.section] = true
			}
			sortProvisions(s.provisions)
		}
	}
}

func findKind(name string) (kind, bool) {
	for _, k := range law {
		if k.name == name {
			return k, true
		}
	}
	return kind{}, false
}

func (k kind) find(name string) (statute, bool) {
	for _, s := range k.statutes {
		if s.name == name {
			return s, true
		}
	}
	return statute{}, false
}

// find looks up a section argument written as §1.1, 1.1, or S.1.1. The
// numbers must match a published section exactly.
func (s statute) find(arg string) (provision, bool) {
	section, ok := resolveSection(arg)
	if !ok {
		return provision{}, false
	}
	for _, p := range s.provisions {
		if p.section == section {
			return p, true
		}
	}
	return provision{}, false
}

// subsections returns the immediate subsections of a section in section
// order: the section, a dot, and one more number. A grandchild is not one.
func (s statute) subsections(section string) []provision {
	var subs []provision
	for _, p := range s.provisions {
		rest, ok := strings.CutPrefix(p.section, section+".")
		if ok && !strings.Contains(rest, ".") {
			subs = append(subs, p)
		}
	}
	return subs
}

// resolveSection turns a section argument into its published § form. The
// argument may carry the section sign, the old S. prefix, or neither.
func resolveSection(arg string) (string, bool) {
	rest, ok := strings.CutPrefix(arg, "§")
	if !ok {
		rest, _ = strings.CutPrefix(arg, "S.")
	}
	if _, ok := parseSection("§" + rest); !ok {
		return "", false
	}
	return "§" + rest, true
}

// parseSection reads the section sign and then one or more positive integers
// separated by dots, with no space and no letter. Zero, a leading zero, a sign,
// an empty part, and a missing section sign are refused.
func parseSection(s string) ([]int, bool) {
	rest, ok := strings.CutPrefix(s, "§")
	if !ok {
		return nil, false
	}
	parts := strings.Split(rest, ".")
	nums := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || p != strconv.Itoa(n) {
			return nil, false
		}
		nums[i] = n
	}
	return nums, true
}

// sectionLess orders numbers component by component. A prefix comes first.
func sectionLess(a, b []int) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

// sortProvisions puts provisions in section order. Every section must parse.
func sortProvisions(ps []provision) {
	sort.SliceStable(ps, func(i, j int) bool {
		a, _ := parseSection(ps[i].section)
		b, _ := parseSection(ps[j].section)
		return sectionLess(a, b)
	})
}
