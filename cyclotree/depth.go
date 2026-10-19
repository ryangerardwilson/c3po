package cyclotree

import "github.com/ryangerardwilson/c3po/ast"

// levels is how many calls can still be opened under a row.
// A call back to a function already on the path adds none.
const maxNavSteps = 100000

type navMemo struct {
	depth int
	down  map[string]bool
}

func (s *screen) bindDepth(n *node) {
	if n == nil || n.cycle || n.id == "" {
		return
	}
	if s.nav == nil {
		s.nav = map[string]navMemo{}
	}
	d, _, _ := s.navRec(n.id, ancestorIDs(n.parent))
	n.levels = d
}

func (s *screen) navRec(id string, anc map[string]bool) (int, map[string]bool, bool) {
	if id == "" || s.calls == nil {
		return 0, nil, true
	}
	s.steps++
	if s.steps > maxNavSteps {
		return 0, nil, false
	}
	if m, ok := s.nav[id]; ok && !hitsAny(m.down, anc) {
		return m.depth, m.down, true
	}
	return s.navKids(id, s.calls.Callees(id), anc)
}

func hitsAny(down, anc map[string]bool) bool {
	for id := range down {
		if anc[id] {
			return true
		}
	}
	return false
}

func (s *screen) navKids(id string, kids []ast.Fn, anc map[string]bool) (int, map[string]bool, bool) {
	best, clean := 0, true
	down := map[string]bool{}
	anc[id] = true
	defer delete(anc, id)
	for _, c := range kids {
		best, clean = s.oneCallee(c, anc, down, best, clean)
	}
	if clean {
		s.nav[id] = navMemo{depth: best, down: down}
	}
	return best, down, clean
}

func (s *screen) oneCallee(c ast.Fn, anc, down map[string]bool, best int, clean bool) (int, bool) {
	if c.ID == "" {
		return best, clean
	}
	down[c.ID] = true
	if anc[c.ID] {
		return best, false
	}
	d, sub, ok := s.navRec(c.ID, anc)
	if d+1 > best {
		best = d + 1
	}
	for k := range sub {
		down[k] = true
	}
	if !ok {
		return best, false
	}
	return best, clean
}
