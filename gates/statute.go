package gates

import (
	_ "embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// gate is one section of a type's statute.
type gate struct {
	section string
	title   string
	body    string
}

// kind is a broad subject and its gates in section order.
type kind struct {
	name  string
	gates []gate
}

//go:embed statute/code/1.md
var codeArchitecture string

// statute is the published legislation, in type order. It is compiled into
// the binary; there is no file, directory, or command that changes it.
var statute = []kind{
	{name: "code", gates: []gate{
		{section: "1", title: "Architecture guardrails", body: codeArchitecture},
	}},
	{name: "essays"},
}

func init() {
	for i := range statute {
		for _, g := range statute[i].gates {
			if _, ok := parseSection(g.section); !ok {
				panic(fmt.Sprintf("gates: bad section %q in %s", g.section, statute[i].name))
			}
		}
		sortGates(statute[i].gates)
	}
}

func findKind(name string) (kind, bool) {
	for _, k := range statute {
		if k.name == name {
			return k, true
		}
	}
	return kind{}, false
}

func (k kind) find(section string) (gate, bool) {
	for _, g := range k.gates {
		if g.section == section {
			return g, true
		}
	}
	return gate{}, false
}

// parseSection reads one or more positive integers separated by dots.
// Zero, a leading zero, a sign, and an empty part are refused.
func parseSection(s string) ([]int, bool) {
	parts := strings.Split(s, ".")
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

// sortGates puts gates in section order. Every section must parse.
func sortGates(gs []gate) {
	sort.SliceStable(gs, func(i, j int) bool {
		a, _ := parseSection(gs[i].section)
		b, _ := parseSection(gs[j].section)
		return sectionLess(a, b)
	})
}
