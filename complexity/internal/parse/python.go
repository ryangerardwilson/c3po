package parse

import (
	"strings"

	"github.com/ryangerardwilson/c3po/complexity/internal/metric"
)

type pyLine struct {
	indent int
	text   string
	line   int
	kind   string
}

type pyState struct {
	cog    int
	nest   int
	cyclo  int
	stack  []int
	events []metric.Event
}

// ParsePython scores def and async def functions using indentation.
func ParsePython(file string, src []byte) ([]metric.Function, []string) {
	lines := pythonLines(string(src))
	code := CodeLines(string(src), "python")
	var fns []metric.Function
	for i := 0; i < len(lines); {
		if isDef(lines[i].text) {
			i = appendDef(file, lines, i, code, &fns)
			continue
		}
		i++
	}
	return fns, nil
}

func appendDef(file string, lines []pyLine, i int, code []bool, out *[]metric.Function) int {
	header := lines[i]
	name, params := defSignature(header.text)
	var body []pyLine
	j := i + 1
	for j < len(lines) {
		if strings.TrimSpace(lines[j].text) == "" {
			j++
			continue
		}
		if lines[j].indent <= header.indent {
			break
		}
		if isDef(lines[j].text) {
			nested := pyLine{indent: lines[j].indent, line: lines[j].line, kind: "nested"}
			j = appendDef(file, lines, j, code, out)
			body = append(body, nested)
			continue
		}
		body = append(body, classify(lines[j]))
		j++
	}
	end := header.line
	if len(body) > 0 {
		end = lastLine(lines, i, j)
	}
	st := scorePy(body)
	*out = append(*out, metric.Function{
		File: file, Line: header.line, EndLine: end, Name: name, Language: "python",
		Cognitive: st.cog, Nesting: st.nest, Lines: countBetween(code, header.line, end),
		Params: params, Cyclomatic: st.cyclo, Events: st.events,
	})
	return j
}

func lastLine(lines []pyLine, start, end int) int {
	line := lines[start].line
	for k := start; k < end && k < len(lines); k++ {
		if lines[k].line > line {
			line = lines[k].line
		}
	}
	return line
}

func isDef(text string) bool {
	t := strings.TrimSpace(text)
	if strings.HasPrefix(t, "async ") {
		t = strings.TrimSpace(strings.TrimPrefix(t, "async"))
	}
	return strings.HasPrefix(t, "def ") || strings.HasPrefix(t, "def(")
}

func defSignature(text string) (string, int) {
	t := strings.TrimSpace(text)
	t = strings.TrimPrefix(t, "async ")
	t = strings.TrimSpace(strings.TrimPrefix(t, "def"))
	name := t
	if i := strings.IndexAny(t, "( "); i >= 0 {
		name = t[:i]
	}
	open := strings.Index(t, "(")
	if open < 0 {
		return name, 0
	}
	return name, pyParamCount(t[open+1 : closeParen(t, open)])
}

func closeParen(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return len(s)
}

func pyParamCount(sig string) int {
	n := 0
	first := true
	for _, part := range splitArgs(sig) {
		name := paramName(part)
		if name == "" || name == "*" || name == "/" {
			continue
		}
		if first && (name == "self" || name == "cls") {
			first = false
			continue
		}
		first = false
		n++
	}
	return n
}

func paramName(part string) string {
	part = strings.TrimSpace(part)
	if k := strings.IndexAny(part, "=:"); k >= 0 {
		part = part[:k]
	}
	return strings.Trim(strings.TrimSpace(part), "*")
}

func splitArgs(sig string) []string {
	var out []string
	depth := 0
	start := 0
	for i := 0; i < len(sig); i++ {
		switch sig[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				out = append(out, sig[start:i])
				start = i + 1
			}
		}
	}
	out = append(out, sig[start:])
	return out
}

func classify(line pyLine) pyLine {
	fields := strings.Fields(line.text)
	if len(fields) == 0 {
		return line
	}
	word := strings.TrimRight(fields[0], ":")
	switch word {
	case "if", "elif", "else", "for", "while", "except", "match", "case", "try", "finally", "with":
		line.kind = word
	default:
		line.kind = "stmt"
	}
	return line
}

func scorePy(body []pyLine) pyState {
	st := pyState{cyclo: 1}
	for _, line := range body {
		st.popTo(line.indent)
		st.add(line)
	}
	return st
}

func (st *pyState) popTo(indent int) {
	for len(st.stack) > 0 && indent <= st.stack[len(st.stack)-1] {
		st.stack = st.stack[:len(st.stack)-1]
	}
}

func (st *pyState) add(line pyLine) {
	ev := metric.Event{Line: line.line, Kind: line.kind, Nesting: len(st.stack)}
	switch line.kind {
	case "if", "for", "while", "except", "match":
		st.score(ev, 1+ev.Nesting, 1, ev.Nesting+1)
		st.bools(line, ev.Nesting)
		st.stack = append(st.stack, line.indent)
	case "elif":
		st.score(ev, 1, 1, 0)
		st.bools(line, ev.Nesting)
		st.stack = append(st.stack, line.indent)
	case "else":
		st.score(ev, 1, 0, 0)
		st.stack = append(st.stack, line.indent)
	case "case":
		st.score(ev, 1, 1, 0)
	case "nested":
		ev.Kind = "nested def"
		st.score(ev, 1+ev.Nesting, 0, 0)
	default:
		st.bools(line, ev.Nesting)
	}
}

func (st *pyState) score(ev metric.Event, cog, cyclo, depth int) {
	ev.Cog, ev.Cyclo, ev.Depth = cog, cyclo, depth
	st.cog += cog
	st.cyclo += cyclo
	if depth > st.nest {
		st.nest = depth
	}
	st.events = append(st.events, ev)
}

func (st *pyState) bools(line pyLine, nesting int) {
	seq, ops, first := boolCost(line.text)
	if ops > 0 {
		st.score(metric.Event{Line: line.line, Kind: first, Nesting: nesting}, seq, ops, 0)
	}
}

func boolCost(text string) (int, int, string) {
	seq, ops := 0, 0
	last, first := "", ""
	for _, w := range words(text) {
		if w != "and" && w != "or" {
			continue
		}
		ops++
		if first == "" {
			first = w
		}
		if w != last {
			seq++
			last = w
		}
	}
	return seq, ops, first
}

func words(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return !(r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})
}

func pythonLines(src string) []pyLine {
	raw := strings.Split(src, "\n")
	out := make([]pyLine, 0, len(raw))
	for i, line := range raw {
		indent, text := splitIndent(blankPy(line))
		out = append(out, pyLine{indent: indent, text: text, line: i + 1})
	}
	return out
}

func splitIndent(s string) (int, string) {
	n := 0
	i := 0
	for i < len(s) {
		if s[i] == ' ' {
			n++
			i++
			continue
		}
		if s[i] == '\t' {
			n += 8
			i++
			continue
		}
		break
	}
	return n, strings.TrimSpace(s[i:])
}

func blankPy(line string) string {
	var b strings.Builder
	for i := 0; i < len(line); {
		if line[i] == '#' {
			break
		}
		if line[i] == '"' || line[i] == '\'' {
			i = skipPyString(line, i)
			b.WriteString(`""`)
			continue
		}
		b.WriteByte(line[i])
		i++
	}
	return b.String()
}

func skipPyString(line string, i int) int {
	q := line[i]
	if i+2 < len(line) && line[i+1] == q && line[i+2] == q {
		end := strings.Index(line[i+3:], strings.Repeat(string(q), 3))
		if end < 0 {
			return len(line)
		}
		return i + 3 + end + 3
	}
	i++
	for i < len(line) {
		if line[i] == '\\' {
			i += 2
			continue
		}
		if line[i] == q {
			return i + 1
		}
		i++
	}
	return i
}
