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

// find looks up a published section exactly, case and all.
func (s statute) find(section string) (provision, bool) {
	for _, p := range s.provisions {
		if p.section == section {
			return p, true
		}
	}
	return provision{}, false
}

// parseSection reads S and then one or more positive integers separated by
// dots. Zero, a leading zero, a sign, an empty part, and a missing S are refused.
func parseSection(s string) ([]int, bool) {
	rest, ok := strings.CutPrefix(s, "S.")
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
