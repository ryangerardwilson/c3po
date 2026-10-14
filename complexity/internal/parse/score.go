package parse

import "github.com/ryangerardwilson/c3po/complexity/internal/metric"

type scorer struct {
	name   string
	cog    int
	nest   int
	cyclo  int
	recur  int
	rust   bool
	inner  map[int]span
	events []metric.Event
}

type scored struct {
	cog, nest, cyclo int
	events           []metric.Event
}

func scoreSpan(toks []Token, s span, inners map[int]span, lang string) scored {
	st := &scorer{name: s.name, cyclo: 1, inner: inners, rust: lang == "rust"}
	st.walk(toks, s.bodyLo, s.bodyHi, 0)
	if st.recur > 0 {
		st.add(metric.Event{Line: st.recur, Kind: "recursion", Cog: 1})
	}
	return scored{st.cog, st.nest, st.cyclo, st.events}
}

func indexInners(all []span) map[int]span {
	m := make(map[int]span, len(all))
	for _, s := range all {
		m[s.start] = s
	}
	return m
}

func (st *scorer) add(e metric.Event) {
	st.cog += e.Cog
	st.cyclo += e.Cyclo
	if e.Depth > st.nest {
		st.nest = e.Depth
	}
	st.events = append(st.events, e)
}

func (st *scorer) walk(toks []Token, lo, hi, nesting int) {
	for i := lo; i < hi; {
		if f, ok := st.inner[i]; ok && f.end > i && f.start >= lo && f.end <= hi {
			st.add(metric.Event{Line: toks[i].Line, Kind: innerKind(f.name), Nesting: nesting, Cog: 1 + nesting})
			i = f.end
			continue
		}
		i = st.one(toks, i, hi, nesting)
	}
}

func innerKind(name string) string {
	if metric.Anonymous(name) {
		return metric.AnonKind(name)
	}
	return "nested function"
}

func skipNL(toks []Token, i, hi int) int {
	for i < hi && toks[i].Lit == "nl" {
		i++
	}
	return i
}

func (st *scorer) one(toks []Token, i, hi, nesting int) int {
	st.noteRecur(toks, i, hi)
	return st.oneLit(toks, i, hi, nesting)
}

func (st *scorer) noteRecur(toks []Token, i, hi int) {
	t := toks[i]
	if st.recur == 0 && t.Kind == "ident" && t.Lit == st.name && i+1 < hi && toks[i+1].Lit == "(" {
		st.recur = t.Line
	}
}

func (st *scorer) oneLit(toks []Token, i, hi, nesting int) int {
	switch toks[i].Lit {
	case "if":
		return st.consumeIf(toks, i, hi, nesting)
	case "for", "while", "loop", "catch":
		return st.consumeLoop(toks, i, hi, nesting)
	case "switch", "match":
		return st.consumeSwitch(toks, i, hi, nesting)
	case "do":
		return st.consumeDo(toks, i, hi, nesting)
	case "&&", "||":
		return st.boolFrom(toks, i, hi, nesting)
	default:
		st.flat(toks[i], nesting)
		return i + 1
	}
}

// flat scores a token that adds points without opening a block.
// Rust has no case or default keyword, and its ? is error propagation.
func (st *scorer) flat(t Token, nesting int) {
	if st.rust {
		return
	}
	switch t.Lit {
	case "case", "default":
		st.add(metric.Event{Line: t.Line, Kind: t.Lit, Nesting: nesting, Cyclo: 1})
	case "?":
		st.add(metric.Event{Line: t.Line, Kind: "ternary", Nesting: nesting, Cog: 1})
	}
}

func (st *scorer) control(t Token, nesting int) {
	st.add(metric.Event{Line: t.Line, Kind: t.Lit, Nesting: nesting, Cog: 1 + nesting, Cyclo: 1, Depth: nesting + 1})
}

func (st *scorer) consumeIf(toks []Token, i, hi, nesting int) int {
	st.control(toks[i], nesting)
	i = st.skipCond(toks, i+1, hi, nesting)
	i = st.scoreStmt(toks, i, hi, nesting+1)
	return st.consumeElse(toks, i, hi, nesting)
}

func (st *scorer) consumeElse(toks []Token, i, hi, nesting int) int {
	i = skipNL(toks, i, hi)
	for st.elseIf(toks, i, hi) {
		st.add(metric.Event{Line: toks[i].Line, Kind: "else if", Nesting: nesting, Cog: 1, Cyclo: 1})
		next := skipNL(toks, i+1, hi)
		i = st.skipCond(toks, next+1, hi, nesting)
		i = st.scoreStmt(toks, i, hi, nesting+1)
		i = skipNL(toks, i, hi)
	}
	if i < hi && toks[i].Lit == "else" {
		st.add(metric.Event{Line: toks[i].Line, Kind: "else", Nesting: nesting, Cog: 1})
		i = st.scoreStmt(toks, i+1, hi, nesting+1)
	}
	return i
}

func (st *scorer) elseIf(toks []Token, i, hi int) bool {
	if i >= hi || toks[i].Lit != "else" {
		return false
	}
	j := skipNL(toks, i+1, hi)
	return j < hi && toks[j].Lit == "if"
}

func (st *scorer) consumeLoop(toks []Token, i, hi, nesting int) int {
	st.control(toks[i], nesting)
	i = st.skipCond(toks, i+1, hi, nesting)
	return st.scoreStmt(toks, i, hi, nesting+1)
}

func (st *scorer) consumeDo(toks []Token, i, hi, nesting int) int {
	st.control(toks[i], nesting)
	i = st.scoreStmt(toks, i+1, hi, nesting+1)
	if i < hi && toks[i].Lit == "while" {
		i = st.skipCond(toks, i+1, hi, nesting)
		if i < hi && toks[i].Lit == ";" {
			i++
		}
	}
	return i
}

func (st *scorer) consumeSwitch(toks []Token, i, hi, nesting int) int {
	head := toks[i]
	st.add(metric.Event{Line: head.Line, Kind: head.Lit, Nesting: nesting, Cog: 1 + nesting, Depth: nesting + 1})
	i = st.skipCond(toks, i+1, hi, nesting)
	if i >= hi || toks[i].Lit != "{" {
		return i
	}
	end := matchPair(toks, i, "{", "}")
	if end < 0 || end > hi {
		end = hi
	}
	if st.rust && head.Lit == "match" {
		st.matchArms(toks, i, end, nesting+1)
	}
	st.walk(toks, i+1, end, nesting+1)
	return end + 1
}

// matchArms adds one cyclomatic point per arm of a Rust match. An arm is a
// => at the top level of the match body.
func (st *scorer) matchArms(toks []Token, open, end, nesting int) {
	o := toks[open]
	for j := open + 1; j < end; j++ {
		t := toks[j]
		if t.Lit == "=>" && t.Brace == o.Brace+1 && t.Paren == o.Paren && t.Brack == o.Brack {
			st.add(metric.Event{Line: t.Line, Kind: "match arm", Nesting: nesting, Cyclo: 1})
		}
	}
}

func (st *scorer) scoreStmt(toks []Token, i, hi, nesting int) int {
	i = skipNL(toks, i, hi)
	if i >= hi {
		return i
	}
	if toks[i].Lit == "{" {
		end := matchPair(toks, i, "{", "}")
		if end < 0 || end > hi {
			end = hi
		}
		st.walk(toks, i+1, end, nesting)
		if end < hi {
			return end + 1
		}
		return end
	}
	if controlWord(toks[i].Lit) {
		return st.one(toks, i, hi, nesting)
	}
	return st.skipSimple(toks, i, hi, nesting)
}

func controlWord(s string) bool {
	switch s {
	case "if", "for", "while", "loop", "switch", "match", "do", "catch":
		return true
	default:
		return false
	}
}

func (st *scorer) skipCond(toks []Token, i, hi, nesting int) int {
	if i >= hi {
		return i
	}
	if toks[i].Lit == "(" {
		end := matchPair(toks, i, "(", ")")
		if end < 0 || end >= hi {
			return hi
		}
		st.addBool(toks[i:end], nesting)
		return end + 1
	}
	if toks[i].Lit == "{" {
		return i
	}
	base := toks[i].Brace
	j := i
	for j < hi {
		if toks[j].Lit == "{" && toks[j].Brace == base {
			st.addBool(toks[i:j], nesting)
			return j
		}
		j++
	}
	st.addBool(toks[i:hi], nesting)
	return hi
}

func (st *scorer) skipSimple(toks []Token, i, hi, nesting int) int {
	paren := 0
	for i < hi {
		lit := toks[i].Lit
		paren = bumpParen(paren, lit)
		if lit == "&&" || lit == "||" {
			i = st.boolFrom(toks, i, hi, nesting)
			continue
		}
		if paren == 0 && (lit == ";" || lit == "nl") {
			return i + 1
		}
		i++
	}
	return i
}

func bumpParen(paren int, lit string) int {
	if lit == "(" {
		return paren + 1
	}
	if lit == ")" && paren > 0 {
		return paren - 1
	}
	return paren
}

// boolFrom scores one run of the same boolean operator. A different
// operator starts a new run.
func (st *scorer) boolFrom(toks []Token, i, hi, nesting int) int {
	op := toks[i].Lit
	ev := metric.Event{Line: toks[i].Line, Kind: op, Nesting: nesting, Cog: 1, Cyclo: 1}
	for i++; i < hi && !breaksBool(toks[i].Lit); i++ {
		lit := toks[i].Lit
		if lit == op {
			ev.Cyclo++
			continue
		}
		if lit == "&&" || lit == "||" {
			st.add(ev)
			return st.boolFrom(toks, i, hi, nesting)
		}
	}
	st.add(ev)
	return i
}

func breaksBool(lit string) bool {
	switch lit {
	case ";", "{", "}", "if", "for", "while", "switch", "nl", "=>":
		return true
	default:
		return false
	}
}

func (st *scorer) addBool(toks []Token, nesting int) {
	var ev metric.Event
	for _, t := range toks {
		if t.Lit != "&&" && t.Lit != "||" {
			if breaksBool(t.Lit) {
				st.flush(&ev)
			}
			continue
		}
		if t.Lit != ev.Kind {
			st.flush(&ev)
			ev = metric.Event{Line: t.Line, Kind: t.Lit, Nesting: nesting, Cog: 1}
		}
		ev.Cyclo++
	}
	st.flush(&ev)
}

func (st *scorer) flush(ev *metric.Event) {
	if ev.Kind != "" {
		st.add(*ev)
	}
	*ev = metric.Event{}
}
