package trace

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ryangerardwilson/c3po/ast/internal/pyimport"
)

// Outline is one named symbol in a file, shared by map, show, and trace.
type Outline struct {
	Kind string
	Name string
	Line int
	End  int
}

// FileOutline lists symbols with the same identity the tracer uses.
func FileOutline(abs, rel string) []Outline {
	ix := &index{byID: map[string]symbol{}}
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".py":
		ix.indexPy(rel, abs)
	case ".ts", ".tsx", ".js", ".jsx":
		ix.indexTS(rel, abs)
	default:
		return nil
	}
	var out []Outline
	for _, s := range ix.syms {
		end := s.end
		if end < s.line {
			end = s.line
		}
		out = append(out, Outline{Kind: s.kind, Name: s.name, Line: s.line, End: end})
	}
	return out
}

func (ix *index) indexPy(rel, abs string) {
	b, err := os.ReadFile(abs)
	if err != nil || strings.Contains(string(b), "\x00") {
		return
	}
	lines := strings.Split(string(b), "\n")
	type blk struct {
		id, name, class string
		indent, start   int
		depth           int
		header          bool
	}
	var stack []blk
	setEnd := func(id string, line int) {
		if s, ok := ix.byID[id]; ok {
			s.end = line
			ix.byID[id] = s
			for i := range ix.syms {
				if ix.syms[i].id == id {
					ix.syms[i].end = line
				}
			}
		}
	}
	popTo := func(ind, line int) {
		for len(stack) > 0 && !stack[len(stack)-1].header && ind <= stack[len(stack)-1].indent {
			top := stack[len(stack)-1]
			setEnd(top.id, line-1)
			stack = stack[:len(stack)-1]
		}
	}
	pushFn := func(trim string, ind, line int, name string) {
		qual, kind := name, "def"
		class := ""
		if len(stack) > 0 {
			top := stack[len(stack)-1]
			if top.class != "" && top.name == top.class {
				qual = top.class + "." + name
				kind = "method"
				class = top.class
			} else if top.name != "" {
				qual = top.name + "." + name
				class = top.class
			}
		}
		s := ix.addSym(rel, qual, kind, line)
		ix.noteDecorators(rel, lines, line-1, s)
		depth, closed, tail := advanceHeader(trim, 0)
		ix.noteFramework(rel, trim, line, s.id, s.name, line)
		stack = append(stack, blk{
			id: s.id, name: qual, class: class, indent: ind, start: line,
			depth: depth, header: !closed,
		})
		if closed && tail != "" && tail != "..." {
			ix.recordPyCalls(rel, tail, line, s.id, s.name, class, line)
		}
	}
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		ind := indentOf(line)
		if len(stack) > 0 && stack[len(stack)-1].header {
			top := stack[len(stack)-1]
			ix.noteFramework(rel, trim, i+1, top.id, top.name, top.start)
			depth, closed, _ := advanceHeader(trim, top.depth)
			top.depth = depth
			top.header = !closed
			stack[len(stack)-1] = top
			continue
		}
		popTo(ind, i+1)
		if m := rePyClass.FindStringSubmatch(trim); len(m) == 2 {
			s := ix.addSym(rel, m[1], "class", i+1)
			depth, closed, _ := advanceHeader(trim, 0)
			stack = append(stack, blk{
				id: s.id, name: m[1], class: m[1], indent: ind, start: i + 1,
				depth: depth, header: !closed,
			})
			continue
		}
		if m := rePyDef.FindStringSubmatch(trim); len(m) == 2 {
			pushFn(trim, ind, i+1, m[1])
			continue
		}
		if len(stack) == 0 {
			ix.notePyImport(rel, trim)
			continue
		}
		owner := stack[len(stack)-1]
		if owner.class != "" && owner.name == owner.class {
			continue
		}
		ix.recordPyCalls(rel, trim, i+1, owner.id, owner.name, owner.class, owner.start)
	}
	popTo(-1, len(lines)+1)
}

// advanceHeader tracks (), [], and {} and reports when a suite header's colon is reached.
func advanceHeader(line string, depth int) (int, bool, string) {
	inStr := byte(0)
	triple := false
	closedAt := -1
	i := 0
	for i < len(line) {
		c := line[i]
		if inStr != 0 {
			if triple {
				if i+2 < len(line) && line[i] == inStr && line[i+1] == inStr && line[i+2] == inStr {
					inStr = 0
					triple = false
					i += 3
					continue
				}
			} else if c == inStr {
				inStr = 0
			} else if c == '\\' && i+1 < len(line) {
				i += 2
				continue
			}
			i++
			continue
		}
		if c == '#' {
			break
		}
		if c == '"' || c == '\'' {
			if i+2 < len(line) && line[i+1] == c && line[i+2] == c {
				inStr = c
				triple = true
				i += 3
				continue
			}
			inStr = c
			i++
			continue
		}
		switch c {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 {
				closedAt = i
			}
		}
		i++
	}
	if closedAt < 0 {
		return depth, false, ""
	}
	return depth, true, strings.TrimSpace(line[closedAt+1:])
}

func (ix *index) recordPyCalls(rel, trim string, line int, id, name, class string, start int) {
	_ = class
	ix.noteFramework(rel, trim, line, id, name, start)
	for _, m := range reSelCall.FindAllStringSubmatch(trim, -1) {
		ix.edges = append(ix.edges, rawEdge{
			fromID: id, fromPath: rel, fromName: name, fromLine: start, callLine: line,
			callee: m[1] + "." + m[2], kind: "selector", confidence: "unresolved",
		})
	}
	for _, loc := range reCall.FindAllStringSubmatchIndex(trim, -1) {
		if loc[2] >= 5 {
			pref := trim[:loc[2]]
			if strings.HasSuffix(pref, "self.") || strings.HasSuffix(pref, "cls.") {
				continue
			}
		}
		callee := trim[loc[2]:loc[3]]
		if keyword(callee) || frameworkCallee(callee) || (isBuiltin(callee) && ix.findInFile(rel, callee).id == "") {
			continue
		}
		target, n := ix.uniqueInFile(rel, callee)
		e := rawEdge{fromID: id, fromPath: rel, fromName: name, fromLine: start, callLine: line, callee: callee}
		if n == 1 {
			e.toID, e.toPath, e.toName, e.toLine = target.id, target.path, target.name, target.line
			e.kind, e.confidence = "direct", "statically-confirmed"
		} else {
			e.kind, e.confidence = "direct", "unresolved"
		}
		ix.edges = append(ix.edges, e)
	}
}

func (ix *index) notePyImport(rel, trim string) {
	imp, ok := pyimport.From(trim)
	if !ok {
		return
	}
	for _, n := range imp.Names {
		ix.imports = append(ix.imports, pyImport{file: rel, local: n.Local, module: imp.Module, orig: n.Orig})
	}
}

func (ix *index) noteDecorators(rel string, lines []string, defLine int, fn symbol) {
	for i := defLine - 1; i >= 0; i-- {
		trim := strings.TrimSpace(lines[i])
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if !strings.HasPrefix(trim, "@") {
			break
		}
		if m := reRoute.FindStringSubmatch(trim); len(m) == 3 {
			label := "route " + strings.ToUpper(m[1]) + " " + m[2]
			ix.edges = append(ix.edges, rawEdge{
				fromID: rel + "#" + label, fromPath: rel, fromName: label, fromLine: i + 1,
				toID: fn.id, toPath: fn.path, toName: fn.name, toLine: fn.line,
				callLine: i + 1, callee: fn.name, kind: "route", confidence: "framework-inferred",
			})
		}
	}
}

func (ix *index) resolvePyMethods() {
	for i := range ix.edges {
		e := &ix.edges[i]
		if e.kind != "selector" || e.toID != "" {
			continue
		}
		if !strings.HasPrefix(e.callee, "self.") && !strings.HasPrefix(e.callee, "cls.") {
			continue
		}
		dot := strings.LastIndex(e.callee, ".")
		if dot < 0 {
			continue
		}
		from, ok := ix.byID[e.fromID]
		if !ok {
			continue
		}
		class := from.name
		if j := strings.LastIndex(class, "."); j >= 0 {
			class = class[:j]
		}
		target := ix.findInFile(e.fromPath, class+"."+e.callee[dot+1:])
		if target.id == "" {
			continue
		}
		e.toID, e.toPath, e.toName, e.toLine = target.id, target.path, target.name, target.line
		e.kind, e.confidence = "method", "statically-confirmed"
	}
}

func (ix *index) resolvePyImports() {
	for i := range ix.edges {
		e := &ix.edges[i]
		if e.kind != "direct" || e.confidence != "unresolved" || e.toID != "" {
			continue
		}
		for _, imp := range ix.imports {
			if imp.file != e.fromPath || imp.local != e.callee {
				continue
			}
			target, n := ix.findImported(imp)
			if n == 1 {
				e.toID, e.toPath, e.toName, e.toLine = target.id, target.path, target.name, target.line
				e.kind, e.confidence = "import", "statically-confirmed"
			}
		}
	}
}

func (ix *index) findImported(imp pyImport) (symbol, int) {
	var hit symbol
	n := 0
	modPath := strings.ReplaceAll(imp.module, ".", "/")
	modPath = strings.TrimPrefix(modPath, "/")
	for _, s := range ix.syms {
		if s.short != imp.orig && s.name != imp.orig {
			continue
		}
		if !pyModuleFile(s.path, eDir(imp.file), modPath) {
			continue
		}
		hit = s
		n++
	}
	return hit, n
}

func eDir(file string) string {
	d := filepath.ToSlash(filepath.Dir(file))
	if d == "." {
		return ""
	}
	return d
}

func pyModuleFile(symPath, fromDir, mod string) bool {
	symPath = filepath.ToSlash(symPath)
	cands := []string{}
	if strings.HasPrefix(mod, ".") {
		// relative: count dots
		return strings.HasSuffix(symPath, strings.TrimLeft(mod, ".")+".py") || strings.Contains(symPath, "/"+strings.TrimLeft(mod, ".")+".py")
	}
	cands = append(cands, mod+".py", mod+"/__init__.py")
	if fromDir != "" {
		cands = append(cands, fromDir+"/"+mod+".py")
	}
	for _, c := range cands {
		if symPath == c || strings.HasSuffix(symPath, "/"+c) {
			return true
		}
	}
	return false
}

func (ix *index) indexTS(rel, abs string) {
	b, err := os.ReadFile(abs)
	if err != nil || strings.Contains(string(b), "\x00") {
		return
	}
	lines := strings.Split(string(b), "\n")
	type fn struct {
		id, name   string
		start, end int
	}
	var fns []fn
	for i := range lines {
		name := tsFuncName(lines, i)
		if name == "" {
			continue
		}
		end := tsFuncEnd(lines, i)
		if end < i+1 {
			end = i + 1
		}
		s := ix.addSym(rel, name, "function", i+1)
		s.end = end
		ix.byID[s.id] = s
		for k := range ix.syms {
			if ix.syms[k].id == s.id {
				ix.syms[k].end = end
			}
		}
		fns = append(fns, fn{id: s.id, name: name, start: i + 1, end: end})
	}
	for i := range lines {
		title, end, ok := testCallbackSpan(lines, i)
		if !ok {
			continue
		}
		s := ix.addSym(rel, title, "function", i+1)
		s.end = end
		ix.byID[s.id] = s
		for k := range ix.syms {
			if ix.syms[k].id == s.id {
				ix.syms[k].end = end
			}
		}
		fns = append(fns, fn{id: s.id, name: title, start: i + 1, end: end})
	}
	ownerAt := func(line int) *fn {
		var best *fn
		for i := range fns {
			f := &fns[i]
			if line < f.start || line > f.end {
				continue
			}
			if best == nil || (f.end-f.start) < (best.end-best.start) {
				best = f
			}
		}
		return best
	}
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "//") {
			continue
		}
		own := ownerAt(i + 1)
		if own == nil {
			continue
		}
		for _, m := range reJSX.FindAllStringSubmatch(trim, -1) {
			ix.addTSCall(rel, own.id, own.name, own.start, i+1, m[1], "jsx")
		}
		if i+1 == own.start {
			continue
		}
		for _, loc := range reCall.FindAllStringSubmatchIndex(trim, -1) {
			callee := trim[loc[2]:loc[3]]
			if keyword(callee) || callee == own.name || jsBuiltin(callee) || (isBuiltin(callee) && ix.findInFile(rel, callee).id == "") {
				continue
			}
			ix.addTSCall(rel, own.id, own.name, own.start, i+1, callee, "direct")
		}
	}
}

func (ix *index) addTSCall(rel, id, name string, start, line int, callee, kind string) {
	target, n := ix.uniqueInFile(rel, callee)
	e := rawEdge{fromID: id, fromPath: rel, fromName: name, fromLine: start, callLine: line, callee: callee, kind: kind, confidence: "unresolved"}
	if n == 1 {
		e.toID, e.toPath, e.toName, e.toLine = target.id, target.path, target.name, target.line
		e.confidence = "statically-confirmed"
	}
	ix.edges = append(ix.edges, e)
}

func symbolKeys(s symbol) []string {
	if s.name == "" || s.name == s.short {
		return []string{s.short}
	}
	return []string{s.short, s.name}
}

func jsBuiltin(name string) bool {
	switch name {
	case "Object", "console", "Math", "JSON", "Promise", "Array", "Number", "String", "Boolean", "Date", "Error", "Symbol", "Reflect", "document", "window", "fetch":
		return true
	default:
		return false
	}
}

func tsFuncName(lines []string, i int) string {
	s := strings.TrimSpace(lines[i])
	if m := reTSFunc.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	if m := reTSConstStart.FindStringSubmatch(s); len(m) == 2 && constIsArrowFunction(lines, i) {
		return m[1]
	}
	return ""
}

// constIsArrowFunction is true only when the initializer itself is an arrow.
// useMemo(() => ...), renderToStaticMarkup(<Tag />), and a later arrow do not count.
func constIsArrowFunction(lines []string, start int) bool {
	eq := strings.Index(lines[start], "=")
	if eq < 0 {
		return false
	}
	return initializerIsArrow(lines, start, eq+1)
}

func initializerIsArrow(lines []string, line, col int) bool {
	i, j := line, col
	skip := func() {
		for i < len(lines) {
			for j < len(lines[i]) {
				c := lines[i][j]
				if c == ' ' || c == '\t' {
					j++
					continue
				}
				if c == '/' && j+1 < len(lines[i]) && lines[i][j+1] == '/' {
					break
				}
				return
			}
			i++
			j = 0
		}
	}
	skip()
	if i >= len(lines) {
		return false
	}
	if strings.HasPrefix(lines[i][j:], "async") && (j+5 == len(lines[i]) || !isIdentByte(lines[i][j+5])) {
		j += 5
		skip()
		if i >= len(lines) {
			return false
		}
	}
	if lines[i][j] == '(' {
		depth := 0
		for i < len(lines) {
			for j < len(lines[i]) {
				c := lines[i][j]
				if c == '"' || c == '\'' || c == '`' {
					q := c
					j++
					for j < len(lines[i]) && lines[i][j] != q {
						if lines[i][j] == '\\' && j+1 < len(lines[i]) {
							j++
						}
						j++
					}
					j++
					continue
				}
				if c == '(' {
					depth++
				} else if c == ')' {
					depth--
					if depth == 0 {
						j++
						skip()
						return i < len(lines) && strings.HasPrefix(lines[i][j:], "=>")
					}
				}
				j++
			}
			i++
			j = 0
		}
		return false
	}
	if isIdentByte(lines[i][j]) {
		for j < len(lines[i]) && isIdentByte(lines[i][j]) {
			j++
		}
		skip()
		return i < len(lines) && strings.HasPrefix(lines[i][j:], "=>")
	}
	return false
}

func isIdentByte(c byte) bool {
	return c == '_' || c == '$' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

var reItCallback = regexp.MustCompile(`^(?:it|test)\s*\(\s*["'\x60]([^"'\x60]+)["'\x60]`)

func testCallbackSpan(lines []string, start int) (string, int, bool) {
	m := reItCallback.FindStringSubmatch(strings.TrimSpace(lines[start]))
	if m == nil {
		return "", 0, false
	}
	end := callbackEnd(lines, start)
	if end < start+1 {
		return "", 0, false
	}
	title := m[1]
	if len(title) > 80 {
		title = title[:80]
	}
	return title, end, true
}

func callbackEnd(lines []string, start int) int {
	idx := strings.Index(lines[start], "=>")
	from := idx
	if idx >= 0 {
		from = idx + 2
	} else if fn := strings.Index(lines[start], "function"); fn >= 0 {
		from = fn + len("function")
	} else {
		return start + 1
	}
	paren, brace := 0, 0
	seenBrace := false
	started := false
	for i := start; i < len(lines); i++ {
		line := lines[i]
		j := 0
		if i == start {
			j = from
		}
		for j < len(line) {
			c := line[j]
			if c == '"' || c == '\'' || c == '`' {
				q := c
				j++
				for j < len(line) && line[j] != q {
					if line[j] == '\\' && j+1 < len(line) {
						j++
					}
					j++
				}
				if j < len(line) {
					j++
				}
				continue
			}
			if !started {
				if c == ' ' || c == '\t' {
					j++
					continue
				}
				if c == '{' {
					seenBrace = true
					brace = 1
					started = true
					j++
					continue
				}
				if c == '(' {
					paren = 1
					started = true
					j++
					continue
				}
				return i + 1
			}
			switch c {
			case '{':
				if paren == 0 {
					brace++
				}
			case '}':
				if seenBrace && paren == 0 {
					brace--
					if brace == 0 {
						return i + 1
					}
				}
			case '(':
				paren++
			case ')':
				if paren > 0 {
					paren--
					if paren == 0 && !seenBrace {
						return i + 1
					}
				}
			}
			j++
		}
	}
	return len(lines)
}

// tsFuncEnd returns the inclusive 1-based end line of a function that starts at lines[start].
// Multiline signatures are included. A brace body and a parenthesized arrow body both count.
func tsFuncEnd(lines []string, start int) int {
	paren, brace := 0, 0
	seenBrace := false
	seenArrow := false
	bodyParen := false
	for i := start; i < len(lines); i++ {
		line := lines[i]
		for idx := 0; idx < len(line); idx++ {
			c := line[idx]
			if c == '/' && idx+1 < len(line) && line[idx+1] == '/' {
				break
			}
			switch c {
			case '(':
				paren++
				if seenArrow && !seenBrace && paren == 1 {
					bodyParen = true
				}
			case ')':
				if paren > 0 {
					paren--
				}
				if bodyParen && paren == 0 && !seenBrace {
					return i + 1
				}
			case '{':
				if paren == 0 {
					seenBrace = true
					brace++
				}
			case '}':
				if seenBrace && paren == 0 {
					brace--
					if brace == 0 {
						return i + 1
					}
				}
			case '=':
				if idx+1 < len(line) && line[idx+1] == '>' && paren == 0 {
					seenArrow = true
					idx++
				}
			}
		}
	}
	if start+1 < len(lines) {
		return len(lines)
	}
	return start + 1
}

func (ix *index) dedupeEdges() {
	rank := map[string]int{"statically-confirmed": 3, "framework-inferred": 2, "interface-inferred": 2, "unresolved": 1}
	best := map[string]rawEdge{}
	var order []string
	for _, e := range ix.edges {
		short := e.callee
		if i := strings.LastIndex(short, "."); i >= 0 {
			short = short[i+1:]
		}
		if isBuiltin(short) && e.confidence == "unresolved" {
			continue
		}
		key := e.fromID + "#" + strconv.Itoa(e.callLine) + "#" + short
		if prev, ok := best[key]; ok {
			if rank[e.confidence] > rank[prev.confidence] || (rank[e.confidence] == rank[prev.confidence] && e.kind == "method" && prev.kind != "method") {
				best[key] = e
			}
			continue
		}
		best[key] = e
		order = append(order, key)
	}
	ix.edges = ix.edges[:0]
	for _, k := range order {
		ix.edges = append(ix.edges, best[k])
	}
}

func isBuiltin(name string) bool {
	switch name {
	case "str", "len", "bool", "int", "float", "list", "dict", "set", "tuple", "print", "append", "get",
		"isinstance", "range", "enumerate", "zip", "map", "filter", "super", "type", "any", "all",
		"min", "max", "sum", "open", "sorted", "reversed", "getattr", "setattr", "hasattr", "format":
		return true
	default:
		return false
	}
}

var (
	reRoute    = regexp.MustCompile(`@(?:\w+\.)?(get|post|put|delete|patch|api_route)\(\s*["']([^"']+)["']`)
	reDepends  = regexp.MustCompile(`\bDepends\(\s*([A-Za-z_][A-Za-z0-9_\.]*)`)
	reInclude  = regexp.MustCompile(`\.include_router\(\s*([A-Za-z_][A-Za-z0-9_\.]*)`)
	reAddRoute = regexp.MustCompile(`\.add_api_route\(\s*["']([^"']+)["']\s*,\s*([A-Za-z_][A-Za-z0-9_\.]*)`)
	reCallback = regexp.MustCompile(`\.(?:add_event_handler|on_event)\(\s*["'][^"']*["']\s*,\s*([A-Za-z_][A-Za-z0-9_\.]*)`)
	reRegister = regexp.MustCompile(`\.(?:register|subscribe)\(\s*([A-Za-z_][A-Za-z0-9_\.]*)\s*\)`)
	reJSX      = regexp.MustCompile(`<([A-Z][A-Za-z0-9_]*)\b`)
)

func frameworkCallee(name string) bool {
	switch name {
	case "Depends", "include_router", "add_api_route", "add_event_handler", "on_event", "register", "subscribe":
		return true
	default:
		return false
	}
}

func (ix *index) noteFramework(rel, trim string, line int, id, name string, start int) {
	for _, m := range reDepends.FindAllStringSubmatch(trim, -1) {
		ix.addInferred(id, rel, name, start, line, m[1], "dependency", "framework-inferred")
	}
	for _, m := range reInclude.FindAllStringSubmatch(trim, -1) {
		ix.addInferred(id, rel, name, start, line, m[1], "router", "configuration-inferred")
	}
	for _, m := range reAddRoute.FindAllStringSubmatch(trim, -1) {
		label := "route API " + m[1]
		ix.edges = append(ix.edges, rawEdge{
			fromID: rel + "#" + label, fromPath: rel, fromName: label, fromLine: line,
			callLine: line, callee: m[2], kind: "route", confidence: "framework-inferred",
		})
	}
	for _, m := range reCallback.FindAllStringSubmatch(trim, -1) {
		ix.addInferred(id, rel, name, start, line, m[1], "callback", "framework-inferred")
	}
	for _, m := range reRegister.FindAllStringSubmatch(trim, -1) {
		ix.addInferred(id, rel, name, start, line, m[1], "registry", "configuration-inferred")
	}
}

func (ix *index) addInferred(id, rel, name string, start, line int, callee, kind, confidence string) {
	ix.edges = append(ix.edges, rawEdge{
		fromID: id, fromPath: rel, fromName: name, fromLine: start,
		callLine: line, callee: callee, kind: kind, confidence: confidence,
	})
}

func (ix *index) resolveFrameworkBinds() {
	counts := map[string]int{}
	one := map[string]symbol{}
	for _, s := range ix.syms {
		if s.kind == "class" {
			continue
		}
		for _, key := range symbolKeys(s) {
			counts[key]++
			one[key] = s
		}
	}
	for i := range ix.edges {
		e := &ix.edges[i]
		if e.toID != "" || (e.confidence != "framework-inferred" && e.confidence != "configuration-inferred") {
			continue
		}
		key := e.callee
		if dot := strings.LastIndex(key, "."); dot >= 0 {
			key = key[dot+1:]
		}
		s, ok := one[e.callee]
		if !ok || counts[e.callee] != 1 {
			s, ok = one[key]
			if !ok || counts[key] != 1 {
				continue
			}
		}
		e.toID, e.toPath, e.toName, e.toLine = s.id, s.path, s.name, s.line
	}
}

type pyImport struct {
	file, local, module, orig string
}

func (ix *index) resolveUniqueNames() {
	counts := map[string]int{}
	one := map[string]symbol{}
	for _, s := range ix.syms {
		if s.kind == "class" {
			continue
		}
		for _, key := range symbolKeys(s) {
			counts[key]++
			one[key] = s
		}
	}
	for i := range ix.edges {
		e := &ix.edges[i]
		if e.toID != "" || e.confidence != "unresolved" {
			continue
		}
		if counts[e.callee] != 1 {
			continue
		}
		s := one[e.callee]
		e.toID, e.toPath, e.toName, e.toLine = s.id, s.path, s.name, s.line
		e.confidence = "statically-confirmed"
	}
}
