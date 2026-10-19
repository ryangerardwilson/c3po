package parse

// Token is one lexical unit. Brace, Paren, and Brack are the depths
// before this token.
type Token struct {
	Kind  string
	Lit   string
	Line  int
	Brace int
	Paren int
	Brack int
}

type lexer struct {
	src   string
	lang  string
	i     int
	line  int
	brace int
	paren int
	brack int
	out   []Token
	inter []int
}

func lex(src, lang string) []Token {
	l := &lexer{src: src, lang: lang, line: 1}
	l.run()
	return l.out
}

func (l *lexer) run() {
	for l.i < len(l.src) {
		if l.comment() || l.space() || l.newline() || l.string() || l.number() || l.ident() || l.operator() {
			continue
		}
		l.punct()
	}
}

func (l *lexer) peek() byte {
	if l.i >= len(l.src) {
		return 0
	}
	return l.src[l.i]
}

func (l *lexer) space() bool {
	c := l.peek()
	if c != ' ' && c != '\t' && c != '\r' {
		return false
	}
	l.i++
	return true
}

func (l *lexer) newline() bool {
	if l.peek() != '\n' {
		return false
	}
	l.emit("nl", "nl")
	l.line++
	l.i++
	return true
}

func (l *lexer) comment() bool {
	if l.i+1 >= len(l.src) {
		return false
	}
	if l.lineComment() {
		return true
	}
	return l.blockComment()
}

func (l *lexer) lineComment() bool {
	if l.src[l.i] != '/' || l.src[l.i+1] != '/' {
		return false
	}
	l.i += 2
	for l.i < len(l.src) && l.src[l.i] != '\n' {
		l.i++
	}
	return true
}

func (l *lexer) blockComment() bool {
	if l.src[l.i] != '/' || l.src[l.i+1] != '*' {
		return false
	}
	l.i += 2
	l.skipBlock()
	return true
}

func (l *lexer) skipBlock() {
	for l.i+1 < len(l.src) && !(l.src[l.i] == '*' && l.src[l.i+1] == '/') {
		if l.src[l.i] == '\n' {
			l.line++
		}
		l.i++
	}
	if l.i+1 < len(l.src) {
		l.i += 2
	}
}

func (l *lexer) string() bool {
	if l.rustRaw() || l.rustChar() {
		return true
	}
	c := l.peek()
	if c != '"' && c != '\'' && c != '`' {
		return false
	}
	if c == '`' && l.lang != "rust" {
		l.template()
		return true
	}
	l.i++
	l.skipQuoted(c)
	l.emit("str", "str")
	return true
}

func (l *lexer) skipQuoted(quote byte) {
	for l.i < len(l.src) {
		if l.src[l.i] == '\\' {
			l.i += 2
			continue
		}
		if l.src[l.i] == '\n' {
			l.line++
		}
		if l.src[l.i] == quote {
			l.i++
			return
		}
		l.i++
	}
}

func (l *lexer) template() {
	l.i++
	for l.i < len(l.src) {
		if l.src[l.i] == '\\' {
			l.i += 2
			continue
		}
		if l.src[l.i] == '`' {
			l.i++
			return
		}
		if l.src[l.i] == '$' && l.i+1 < len(l.src) && l.src[l.i+1] == '{' {
			l.i += 2
			l.inter = append(l.inter, l.brace)
			return
		}
		if l.src[l.i] == '\n' {
			l.line++
		}
		l.i++
	}
}

func (l *lexer) rustRaw() bool {
	j, ok := l.rawStart()
	if !ok {
		return false
	}
	hashes := j - (l.i + 1)
	l.i = j + 1
	return l.rawRest(hashes)
}

func (l *lexer) rawStart() (int, bool) {
	if l.lang != "rust" || l.peek() != 'r' {
		return 0, false
	}
	j := l.i + 1
	for j < len(l.src) && l.src[j] == '#' {
		j++
	}
	if j >= len(l.src) || l.src[j] != '"' {
		return 0, false
	}
	return j, true
}

func (l *lexer) rawRest(hashes int) bool {
	for l.i < len(l.src) {
		if l.src[l.i] == '\n' {
			l.line++
		}
		if l.src[l.i] == '"' && l.rawEnd(hashes) {
			l.i += 1 + hashes
			l.emit("str", "str")
			return true
		}
		l.i++
	}
	l.emit("str", "str")
	return true
}

func (l *lexer) rawEnd(hashes int) bool {
	for n := 0; n < hashes; n++ {
		if l.i+1+n >= len(l.src) || l.src[l.i+1+n] != '#' {
			return false
		}
	}
	return true
}

func (l *lexer) rustChar() bool {
	if l.lang != "rust" || l.peek() != '\'' {
		return false
	}
	if l.simpleChar() {
		return true
	}
	if l.escapedChar() {
		return true
	}
	l.i++
	l.identBody()
	return true
}

func (l *lexer) simpleChar() bool {
	if l.i+2 < len(l.src) && l.src[l.i+1] != '\\' && l.src[l.i+2] == '\'' {
		l.i += 3
		l.emit("str", "str")
		return true
	}
	return false
}

func (l *lexer) escapedChar() bool {
	if l.i+1 >= len(l.src) || l.src[l.i+1] != '\\' {
		return false
	}
	l.i += 2
	for l.i < len(l.src) && l.src[l.i] != '\'' {
		l.i++
	}
	if l.i < len(l.src) {
		l.i++
	}
	l.emit("str", "str")
	return true
}

func (l *lexer) number() bool {
	if !digit(l.peek()) {
		return false
	}
	for l.i < len(l.src) && numChar(l.src[l.i]) {
		l.i++
	}
	l.emit("num", "0")
	return true
}

func digit(c byte) bool {
	return c >= '0' && c <= '9'
}

func numChar(c byte) bool {
	if digit(c) || c == '_' || c == '.' {
		return true
	}
	return letter(c)
}

func letter(c byte) bool {
	lower := c >= 'a' && c <= 'z'
	upper := c >= 'A' && c <= 'Z'
	return lower || upper
}

func (l *lexer) ident() bool {
	c := l.peek()
	if !isIdentStart(c) {
		return false
	}
	start := l.i
	l.identBody()
	l.emit("ident", l.src[start:l.i])
	return true
}

func (l *lexer) identBody() {
	for l.i < len(l.src) && isIdentCont(l.src[l.i]) {
		l.i++
	}
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func (l *lexer) operator() bool {
	rest := l.src[l.i:]
	ops := []string{"...", ">>=", "<<=", "=>", "&&", "||", "??", "==", "!=", "<=", ">=", ":=", "+=", "-=", "*=", "/=", "::", "?.", "++", "--", "<<", ">>"}
	for _, op := range ops {
		if len(rest) >= len(op) && rest[:len(op)] == op {
			l.i += len(op)
			l.emit("op", op)
			return true
		}
	}
	return false
}

func (l *lexer) punct() {
	c := l.peek()
	l.emit("punct", string(c))
	switch c {
	case '{':
		l.brace++
	case '}':
		if l.brace > 0 {
			l.brace--
		}
		l.leaveTemplate()
	case '(':
		l.paren++
	case ')':
		if l.paren > 0 {
			l.paren--
		}
	case '[':
		l.brack++
	case ']':
		if l.brack > 0 {
			l.brack--
		}
	}
	l.i++
}

func (l *lexer) leaveTemplate() {
	if len(l.inter) == 0 || l.lang == "rust" {
		return
	}
	if l.brace != l.inter[len(l.inter)-1] {
		return
	}
	l.inter = l.inter[:len(l.inter)-1]
	l.template()
}

func (l *lexer) emit(kind, lit string) {
	l.out = append(l.out, Token{
		Kind: kind, Lit: lit, Line: l.line,
		Brace: l.brace, Paren: l.paren, Brack: l.brack,
	})
}
