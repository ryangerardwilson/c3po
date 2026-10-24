package sheet

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
)

type Kind int

const (
	Empty Kind = iota
	Number
	Text
	Error
)

type Value struct {
	Kind Kind
	Num  float64
	Text string
	Err  string
}

func (v Value) String() string {
	switch v.Kind {
	case Number:
		return formatNumber(v.Num)
	case Text:
		return v.Text
	case Error:
		return v.Err
	default:
		return ""
	}
}

func num(n float64) Value { return Value{Kind: Number, Num: n} }
func txt(s string) Value  { return Value{Kind: Text, Text: s} }
func errv(code string) Value {
	return Value{Kind: Error, Err: code}
}

func formatNumber(n float64) string {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return "#ERR"
	}
	if n == 0 {
		return "0"
	}
	s := strconv.FormatFloat(n, 'f', 6, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	if s == "-0" || s == "" {
		return "0"
	}
	return s
}

func formatGrouped(n float64) string {
	raw := formatNumber(n)
	sign := ""
	if strings.HasPrefix(raw, "-") {
		sign = "-"
		raw = raw[1:]
	}
	intPart, frac, hasFrac := strings.Cut(raw, ".")
	var b strings.Builder
	for i := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteByte(intPart[i])
	}
	if hasFrac {
		b.WriteByte('.')
		b.WriteString(frac)
	}
	return sign + b.String()
}

var errSyntax = errors.New("syntax")

type expr interface {
	eval(s *Sheet, stack map[visit]bool) Value
}

type numberExpr struct{ n float64 }
type cellExpr struct {
	sheet string
	c, r  int
}
type prefixExpr struct {
	op byte
	x  expr
}
type binaryExpr struct {
	op   byte
	l, r expr
}
type callArg struct {
	isRange bool
	sheet   string
	sheet2  string
	c1, r1  int
	c2, r2  int
	x       expr
}
type callExpr struct {
	name string
	args []callArg
}

func (e numberExpr) eval(*Sheet, map[visit]bool) Value { return num(e.n) }

type errorExpr struct{ code string }

func (e errorExpr) eval(*Sheet, map[visit]bool) Value { return errv(e.code) }

func (e cellExpr) eval(s *Sheet, stack map[visit]bool) Value {
	target := s.resolve(e.sheet)
	if target == nil {
		return errv("#REF")
	}
	return target.eval(e.c, e.r, stack)
}

func (e prefixExpr) eval(s *Sheet, stack map[visit]bool) Value {
	n, ev, ok := numberOf(e.x.eval(s, stack))
	if !ok {
		return ev
	}
	if e.op == '-' {
		return num(-n)
	}
	return num(n)
}

func (e binaryExpr) eval(s *Sheet, stack map[visit]bool) Value {
	ln, lv, lok := numberOf(e.l.eval(s, stack))
	if !lok {
		return lv
	}
	rn, rv, rok := numberOf(e.r.eval(s, stack))
	if !rok {
		return rv
	}
	switch e.op {
	case '+':
		return num(ln + rn)
	case '-':
		return num(ln - rn)
	case '*':
		return num(ln * rn)
	case '/':
		if rn == 0 {
			return errv("#DIV/0")
		}
		return num(ln / rn)
	default:
		return errv("#ERR")
	}
}

func (e callExpr) eval(s *Sheet, stack map[visit]bool) Value {
	var nums []float64
	for _, arg := range e.args {
		if arg.isRange {
			if arg.sheet != "" && arg.sheet2 != "" && !strings.EqualFold(arg.sheet, arg.sheet2) {
				return errv("#REF")
			}
			name := arg.sheet
			if name == "" {
				name = arg.sheet2
			}
			target := s.resolve(name)
			if target == nil {
				return errv("#REF")
			}
			got, ev := target.walk(arg.c1, arg.r1, arg.c2, arg.r2, stack)
			if ev.Kind == Error {
				return ev
			}
			nums = append(nums, got...)
			continue
		}
		v := arg.x.eval(s, stack)
		switch v.Kind {
		case Error:
			return v
		case Number:
			nums = append(nums, v.Num)
		case Empty:
		default:
			return errv("#VALUE")
		}
	}
	return reduce(e.name, nums)
}

func reduce(name string, nums []float64) Value {
	switch name {
	case "SUM":
		total := 0.0
		for _, n := range nums {
			total += n
		}
		return num(total)
	case "AVERAGE", "AVG":
		if len(nums) == 0 {
			return errv("#DIV/0")
		}
		total := 0.0
		for _, n := range nums {
			total += n
		}
		return num(total / float64(len(nums)))
	case "COUNT":
		return num(float64(len(nums)))
	case "MIN":
		if len(nums) == 0 {
			return errv("#NUM")
		}
		best := nums[0]
		for _, n := range nums[1:] {
			if n < best {
				best = n
			}
		}
		return num(best)
	case "MAX":
		if len(nums) == 0 {
			return errv("#NUM")
		}
		best := nums[0]
		for _, n := range nums[1:] {
			if n > best {
				best = n
			}
		}
		return num(best)
	case "MEDIAN":
		if len(nums) == 0 {
			return errv("#NUM")
		}
		return num(medianOf(nums))
	case "MODE":
		mode, ok := modeOf(nums)
		if !ok {
			return errv("#N/A")
		}
		return num(mode)
	default:
		return errv("#NAME")
	}
}

func numberOf(v Value) (float64, Value, bool) {
	switch v.Kind {
	case Number:
		return v.Num, Value{}, true
	case Empty:
		return 0, Value{}, true
	case Error:
		return 0, v, false
	default:
		return 0, errv("#VALUE"), false
	}
}

func medianOf(nums []float64) float64 {
	cp := append([]float64(nil), nums...)
	sort.Float64s(cp)
	n := len(cp)
	mid := n / 2
	if n%2 == 1 {
		return cp[mid]
	}
	return (cp[mid-1] + cp[mid]) / 2
}

func modeOf(nums []float64) (float64, bool) {
	counts := map[string]int{}
	value := map[string]float64{}
	for _, n := range nums {
		key := formatNumber(n)
		counts[key]++
		if _, ok := value[key]; !ok {
			value[key] = n
		}
	}
	best := 0
	var mode float64
	for key, count := range counts {
		n := value[key]
		if count > best || (count == best && n < mode) {
			best = count
			mode = n
		}
	}
	if best <= 1 {
		return 0, false
	}
	return mode, true
}

type parser struct {
	s string
	i int
}

func evalFormula(src string, s *Sheet, stack map[visit]bool) Value {
	p := &parser{s: src[1:]}
	ex, err := p.parseExpr()
	if err != nil {
		return errv("#ERR")
	}
	p.skip()
	if p.i != len(p.s) {
		return errv("#ERR")
	}
	return ex.eval(s, stack)
}

func (p *parser) parseExpr() (expr, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	for {
		p.skip()
		op := p.peek()
		if op != '+' && op != '-' {
			break
		}
		p.i++
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		left = binaryExpr{op: op, l: left, r: right}
	}
	return left, nil
}

func (p *parser) parseTerm() (expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		p.skip()
		op := p.peek()
		if op != '*' && op != '/' {
			break
		}
		p.i++
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = binaryExpr{op: op, l: left, r: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (expr, error) {
	p.skip()
	if op := p.peek(); op == '+' || op == '-' {
		p.i++
		x, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return prefixExpr{op: op, x: x}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (expr, error) {
	p.skip()
	switch ch := p.peek(); {
	case ch == '(':
		p.i++
		x, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		p.skip()
		if p.peek() != ')' {
			return nil, errSyntax
		}
		p.i++
		return x, nil
	case ch == '.' || isDigit(ch):
		return p.parseNumber()
	case ch == '\'':
		return p.parseQuotedRef()
	case ch == '$' || isLetter(ch):
		return p.parseIdent()
	case ch == '#':
		return p.parseRefError()
	default:
		return nil, errSyntax
	}
}

func (p *parser) parseRefError() (expr, error) {
	const token = "#REF!"
	if !strings.HasPrefix(p.s[p.i:], token) {
		return nil, errSyntax
	}
	p.i += len(token)
	return errorExpr{code: "#REF"}, nil
}

func (p *parser) parseNumber() (expr, error) {
	start := p.i
	sawDot := false
	sawDigit := false
	for p.i < len(p.s) {
		ch := p.s[p.i]
		switch {
		case isDigit(ch):
			sawDigit = true
			p.i++
		case ch == '.' && !sawDot:
			sawDot = true
			p.i++
		default:
			goto done
		}
	}
done:
	if !sawDigit {
		return nil, errSyntax
	}
	n, err := strconv.ParseFloat(p.s[start:p.i], 64)
	if err != nil {
		return nil, errSyntax
	}
	return numberExpr{n: n}, nil
}

func (p *parser) parseIdent() (expr, error) {
	if sheetName, c, r, end, ok := p.refAt(p.i); ok && sheetName != "" {
		p.i = end
		return cellExpr{sheet: sheetName, c: c, r: r}, nil
	}
	absCol := false
	if p.peek() == '$' {
		absCol = true
		p.i++
	}
	nameStart := p.i
	for isLetter(p.peek()) {
		p.i++
	}
	if p.i == nameStart {
		return nil, errSyntax
	}
	name := p.s[nameStart:p.i]
	p.skip()
	if p.peek() == '(' {
		if absCol {
			return nil, errSyntax
		}
		return p.parseCall(name)
	}
	if p.peek() == '$' {
		p.i++
	}
	rowStart := p.i
	for isDigit(p.peek()) {
		p.i++
	}
	if p.i == rowStart {
		return nil, errSyntax
	}
	row, err := strconv.Atoi(p.s[rowStart:p.i])
	if err != nil || row < 1 {
		return nil, errSyntax
	}
	col, ok := ColIndex(name)
	if !ok {
		return nil, errSyntax
	}
	return cellExpr{c: col, r: row - 1}, nil
}

func (p *parser) parseCall(name string) (expr, error) {
	p.i++
	var args []callArg
	p.skip()
	if p.peek() == ')' {
		p.i++
		return callExpr{name: strings.ToUpper(name), args: args}, nil
	}
	for {
		arg, err := p.parseArg()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		p.skip()
		switch p.peek() {
		case ',':
			p.i++
		case ')':
			p.i++
			return callExpr{name: strings.ToUpper(name), args: args}, nil
		default:
			return nil, errSyntax
		}
	}
}

func (p *parser) parseArg() (callArg, error) {
	if sheetName, c1, r1, end, ok := p.refAt(p.i); ok {
		saved := p.i
		p.i = end
		p.skip()
		if p.peek() == ':' {
			p.i++
			sheet2, c2, r2, end2, ok2 := p.refAt(p.i)
			if !ok2 {
				return callArg{}, errSyntax
			}
			p.i = end2
			return callArg{isRange: true, sheet: sheetName, sheet2: sheet2, c1: c1, r1: r1, c2: c2, r2: r2}, nil
		}
		p.i = saved
	}
	ex, err := p.parseExpr()
	if err != nil {
		return callArg{}, err
	}
	return callArg{x: ex}, nil
}

func (p *parser) parseQuotedRef() (expr, error) {
	sheetName, c, r, end, ok := p.refAt(p.i)
	if !ok || sheetName == "" {
		return nil, errSyntax
	}
	p.i = end
	return cellExpr{sheet: sheetName, c: c, r: r}, nil
}

func (p *parser) refAt(start int) (sheetName string, c, r, end int, ok bool) {
	i := start
	for i < len(p.s) && (p.s[i] == ' ' || p.s[i] == '\t') {
		i++
	}
	if i < len(p.s) && p.s[i] == 39 {
		name, j, good := quotedSheetAt(p.s, i)
		if !good || j >= len(p.s) || p.s[j] != '!' {
			return "", 0, 0, start, false
		}
		c, r, end, ok = p.cellRef(j + 1)
		if !ok {
			return "", 0, 0, start, false
		}
		return name, c, r, end, true
	}
	if i < len(p.s) && isLetter(p.s[i]) {
		j := i + 1
		for j < len(p.s) && bareSheetChar(p.s[j]) {
			j++
		}
		if j < len(p.s) && p.s[j] == '!' {
			c, r, end, ok = p.cellRef(j + 1)
			if ok {
				return p.s[i:j], c, r, end, true
			}
		}
	}
	c, r, end, ok = p.cellRef(i)
	return "", c, r, end, ok
}

func (p *parser) cellRef(start int) (c, r, end int, ok bool) {
	i := start
	for i < len(p.s) && (p.s[i] == ' ' || p.s[i] == '\t') {
		i++
	}
	if i < len(p.s) && p.s[i] == '$' {
		i++
	}
	nameStart := i
	for i < len(p.s) && isLetter(p.s[i]) {
		i++
	}
	if i == nameStart {
		return 0, 0, start, false
	}
	nameEnd := i
	if i < len(p.s) && p.s[i] == '$' {
		i++
	}
	rowStart := i
	for i < len(p.s) && isDigit(p.s[i]) {
		i++
	}
	if i == rowStart {
		return 0, 0, start, false
	}
	col, good := ColIndex(p.s[nameStart:nameEnd])
	if !good {
		return 0, 0, start, false
	}
	row, err := strconv.Atoi(p.s[rowStart:i])
	if err != nil || row < 1 {
		return 0, 0, start, false
	}
	return col, row - 1, i, true
}

func (p *parser) skip() {
	for p.i < len(p.s) && (p.s[p.i] == ' ' || p.s[p.i] == '\t') {
		p.i++
	}
}

func (p *parser) peek() byte {
	if p.i >= len(p.s) {
		return 0
	}
	return p.s[p.i]
}

func isLetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
