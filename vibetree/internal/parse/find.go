package parse

import "fmt"

type span struct {
	name           string
	line, endLine  int
	params         int
	start, end     int
	bodyLo, bodyHi int
	expr           bool
}

func findFunctions(toks []Token, lang string) []span {
	var out []span
	for i := 0; i < len(toks); i++ {
		if s, ok := tryFn(toks, i, lang); ok {
			out = append(out, s)
		}
	}
	return out
}

func tryFn(toks []Token, i int, lang string) (span, bool) {
	if s, ok := tryKeyword(toks, i, lang); ok {
		return s, true
	}
	if s, ok := tryArrow(toks, i, lang); ok {
		return s, true
	}
	return tryMethod(toks, i)
}

func tryKeyword(toks []Token, i int, lang string) (span, bool) {
	lit := toks[i].Lit
	if lit == "function" || (lang == "rust" && lit == "fn") {
		return readKeywordFn(toks, i, lang)
	}
	return span{}, false
}

func tryArrow(toks []Token, i int, lang string) (span, bool) {
	lit := toks[i].Lit
	if lit == "=>" && lang != "rust" {
		return readArrow(toks, i)
	}
	if lang == "rust" && (lit == "|" || lit == "||") && closureStart(toks, i) {
		return readRustClosure(toks, i)
	}
	return span{}, false
}

// closureStart reports whether a | or || at i opens a Rust closure rather
// than an or. An or follows an operand.
func closureStart(toks []Token, i int) bool {
	j := i - 1
	for j >= 0 && toks[j].Lit == "nl" {
		j--
	}
	if j < 0 {
		return true
	}
	t := toks[j]
	switch t.Lit {
	case "move", "return", "async", "in":
		return true
	case ")", "]", "}", "?":
		return false
	}
	return t.Kind != "ident" && t.Kind != "num" && t.Kind != "str"
}

func tryMethod(toks []Token, i int) (span, bool) {
	lit := toks[i].Lit
	if toks[i].Kind == "ident" && !keyword(lit) && methodStart(toks, i) {
		return readMethod(toks, i)
	}
	return span{}, false
}

func keyword(s string) bool {
	switch s {
	case "if", "else", "for", "while", "do", "switch", "catch", "function", "return",
		"var", "let", "const", "class", "new", "try", "throw", "typeof", "instanceof",
		"fn", "match", "loop", "impl", "pub", "mut", "struct", "enum", "use", "mod",
		"async", "await", "yield", "case", "default", "break", "continue", "static",
		"get", "set", "of", "in", "import", "export", "from", "type", "interface":
		return true
	default:
		return false
	}
}

func methodStart(toks []Token, i int) bool {
	prev := i - 1
	for prev >= 0 && toks[prev].Lit == "nl" {
		prev--
	}
	if prev < 0 {
		return true
	}
	switch toks[prev].Lit {
	case "{", "}", ";", ":", ",", "public", "private", "protected", "static", "async",
		"get", "set", "export", "default", "readonly", "pub", "override", "abstract":
		return true
	default:
		return false
	}
}

func readKeywordFn(toks []Token, i int, lang string) (span, bool) {
	j := i + 1
	name := fmt.Sprintf("function@%d", toks[i].Line)
	if lang == "rust" {
		name = "fn"
	}
	if j < len(toks) && toks[j].Kind == "ident" && !keyword(toks[j].Lit) {
		name = toks[j].Lit
		j++
	}
	j = skipAngles(toks, j)
	if j >= len(toks) || toks[j].Lit != "(" {
		return span{}, false
	}
	pend := matchPair(toks, j, "(", ")")
	if pend < 0 {
		return span{}, false
	}
	params := countParams(toks[j+1:pend], lang == "rust")
	k := skipToBrace(toks, pend+1)
	if k < 0 {
		return span{}, false
	}
	bend := matchPair(toks, k, "{", "}")
	if bend < 0 {
		return span{}, false
	}
	return span{name, toks[i].Line, toks[bend].Line, params, i, bend + 1, k + 1, bend, false}, true
}

func readMethod(toks []Token, i int) (span, bool) {
	if i+1 >= len(toks) || toks[i+1].Lit != "(" {
		return span{}, false
	}
	pend := matchPair(toks, i+1, "(", ")")
	if pend < 0 {
		return span{}, false
	}
	k := skipToBrace(toks, pend+1)
	if k < 0 || k > pend+8 {
		return span{}, false
	}
	bend := matchPair(toks, k, "{", "}")
	if bend < 0 {
		return span{}, false
	}
	params := countParams(toks[i+2:pend], false)
	return span{toks[i].Lit, toks[i].Line, toks[bend].Line, params, i, bend + 1, k + 1, bend, false}, true
}

func readArrow(toks []Token, i int) (span, bool) {
	params, ok := paramsBefore(toks, i)
	if !ok {
		return span{}, false
	}
	name := fmt.Sprintf("arrow@%d", toks[i].Line)
	if i+1 >= len(toks) {
		return span{}, false
	}
	if toks[i+1].Lit == "{" {
		bend := matchPair(toks, i+1, "{", "}")
		if bend < 0 {
			return span{}, false
		}
		return span{name, toks[i].Line, toks[bend].Line, params, i, bend + 1, i + 2, bend, false}, true
	}
	end := exprEnd(toks, i+1)
	line := toks[i].Line
	if end > i+1 {
		line = toks[end-1].Line
	}
	return span{name, toks[i].Line, line, params, i, end, i + 1, end, true}, true
}

func readRustClosure(toks []Token, i int) (span, bool) {
	pend := closingPipe(toks, i)
	if pend < 0 {
		return span{}, false
	}
	params := 0
	if pend > i+1 {
		params = countParams(toks[i+1:pend], true)
	}
	name := fmt.Sprintf("closure@%d", toks[i].Line)
	if k := closureBlock(toks, pend+1); k >= 0 {
		bend := matchPair(toks, k, "{", "}")
		if bend < 0 {
			return span{}, false
		}
		return span{name, toks[i].Line, toks[bend].Line, params, i, bend + 1, k + 1, bend, false}, true
	}
	end := exprEnd(toks, pend+1)
	return span{name, toks[i].Line, toks[i].Line, params, i, end, pend + 1, end, true}, true
}

// closingPipe returns the | that closes closure parameters. A zero-argument
// closure is one || token, so it closes itself.
func closingPipe(toks []Token, i int) int {
	if toks[i].Lit == "||" {
		return i
	}
	for j := i + 1; j < len(toks); j++ {
		switch toks[j].Lit {
		case "|":
			return j
		case "{", "}", ";":
			return -1
		}
	}
	return -1
}

// closureBlock returns the { of a block body, after an optional -> type.
func closureBlock(toks []Token, k int) int {
	if k < len(toks) && toks[k].Lit == "{" {
		return k
	}
	if k+1 < len(toks) && toks[k].Lit == "-" && toks[k+1].Lit == ">" {
		return skipToBrace(toks, k+2)
	}
	return -1
}

func paramsBefore(toks []Token, arrow int) (int, bool) {
	j := arrow - 1
	for j >= 0 && (toks[j].Lit == "nl" || toks[j].Lit == "async") {
		j--
	}
	if j < 0 {
		return 0, false
	}
	if toks[j].Kind == "ident" {
		return 1, true
	}
	if toks[j].Lit != ")" {
		return 0, false
	}
	open := matchBack(toks, j, "(", ")")
	if open < 0 {
		return 0, false
	}
	return countParams(toks[open+1:j], false), true
}

func skipAngles(toks []Token, j int) int {
	if j >= len(toks) || toks[j].Lit != "<" {
		return j
	}
	depth := 0
	for j < len(toks) {
		switch toks[j].Lit {
		case "<":
			depth++
		case ">":
			depth--
			if depth == 0 {
				return j + 1
			}
		case "{", ";":
			return j
		}
		j++
	}
	return j
}

func skipToBrace(toks []Token, j int) int {
	if j >= len(toks) {
		return -1
	}
	return scanBrace(toks, j, toks[j].Brace, 0)
}

func scanBrace(toks []Token, j, base, paren int) int {
	for ; j < len(toks); j++ {
		next, done, n := braceAt(toks[j], j, base, paren)
		paren = n
		if done {
			return next
		}
	}
	return -1
}

func braceAt(t Token, j, base, paren int) (int, bool, int) {
	switch t.Lit {
	case "(":
		return 0, false, paren + 1
	case ")":
		return 0, false, decParen(paren)
	case "{":
		if paren == 0 && t.Brace == base {
			return j, true, paren
		}
	case ";":
		if paren == 0 {
			return -1, true, paren
		}
	}
	return 0, false, paren
}

func decParen(paren int) int {
	if paren > 0 {
		return paren - 1
	}
	return paren
}

func exprEnd(toks []Token, j int) int {
	if j >= len(toks) {
		return j
	}
	t := toks[j]
	return exprStop(toks, j, t.Brace, t.Paren, t.Brack)
}

func exprStop(toks []Token, j, brace, paren, brack int) int {
	for j < len(toks) {
		if exprBreak(toks[j], brace, paren, brack) {
			return j
		}
		j++
	}
	return j
}

func exprBreak(t Token, brace, paren, brack int) bool {
	if t.Brace < brace || t.Paren < paren || t.Brack < brack {
		return true
	}
	return t.Brace == brace && t.Paren == paren && exprSep(t.Lit)
}

func exprSep(lit string) bool {
	return lit == ";" || lit == "nl" || lit == ","
}

func matchPair(toks []Token, i int, open, close string) int {
	if i >= len(toks) || toks[i].Lit != open {
		return -1
	}
	depth := 0
	for ; i < len(toks); i++ {
		if toks[i].Lit == open {
			depth++
		}
		if toks[i].Lit == close {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func matchBack(toks []Token, i int, open, close string) int {
	if i >= len(toks) || toks[i].Lit != close {
		return -1
	}
	depth := 0
	for ; i >= 0; i-- {
		if toks[i].Lit == close {
			depth++
		}
		if toks[i].Lit == open {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func countParams(toks []Token, rust bool) int {
	n := 0
	for _, sec := range splitTop(toks) {
		if paramSection(sec, rust) {
			n++
		}
	}
	return n
}

type depths struct{ p, b, k, a int }

func splitTop(toks []Token) [][]Token {
	var out [][]Token
	var cur []Token
	var d depths
	for _, t := range toks {
		if d.comma(t.Lit) {
			out = append(out, cur)
			cur = nil
			continue
		}
		d.bump(t.Lit)
		cur = append(cur, t)
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out
}

func (d depths) comma(lit string) bool {
	return lit == "," && d.p == 0 && d.b == 0 && d.k == 0 && d.a == 0
}

func (d *depths) bump(lit string) {
	switch lit {
	case "(":
		d.p++
	case ")":
		d.dec(&d.p)
	case "[":
		d.b++
	case "]":
		d.dec(&d.b)
	case "{":
		d.k++
	case "}":
		d.dec(&d.k)
	case "<":
		d.openAngle()
	case ">":
		d.dec(&d.a)
	}
}

func (d *depths) dec(n *int) {
	if *n > 0 {
		*n--
	}
}

func (d *depths) openAngle() {
	if d.p == 0 && d.b == 0 && d.k == 0 {
		d.a++
	}
}

func paramSection(sec []Token, rust bool) bool {
	if len(sec) == 0 {
		return false
	}
	if rust && isSelf(sec) {
		return false
	}
	for _, t := range sec {
		if t.Kind == "ident" || t.Lit == "..." {
			return true
		}
	}
	return false
}

func isSelf(sec []Token) bool {
	for _, t := range sec {
		if t.Lit == "self" {
			return true
		}
	}
	return false
}
