package trace

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/measure"
)

const defaultLimit = 200

// Symbol is a function, method, or class the index can name.
type Symbol struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Name  string `json:"name"`
	Short string `json:"short"`
	Kind  string `json:"kind"`
	Line  int    `json:"line"`
}

// Edge is one call, labeled with how sure the binding is.
type Edge struct {
	Confidence string `json:"confidence"`
	Kind       string `json:"kind"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Symbol     string `json:"symbol"`
	PeerFile   string `json:"peer_file,omitempty"`
	PeerLine   int    `json:"peer_line,omitempty"`
	PeerSymbol string `json:"peer_symbol,omitempty"`
	CalleeText string `json:"callee_text"`
	Reason     string `json:"reason,omitempty"`
}

// Assessment says whether this impact result is the whole outcome.
// conclusion is "complete" or "partial". An empty callers or callees
// slice means the search found nothing. A null slice means that side
// was not requested.
type Assessment struct {
	Direction       string   `json:"direction"`
	MaxDepth        int      `json:"max_depth"`
	PathLimit       int      `json:"path_limit"`
	UnresolvedEdges int      `json:"unresolved_edges"`
	CycleCuts       int      `json:"cycle_cuts"`
	Complete        bool     `json:"complete"`
	Conclusion      string   `json:"conclusion"`
	StopReasons     []string `json:"stop_reasons"`
	Unsupported     []string `json:"unsupported"`
}

// Result is a paginated trace. Truncated results are not the whole graph.
type Result struct {
	Query      string      `json:"query"`
	Mode       string      `json:"mode"`
	Resolved   Symbol      `json:"resolved"`
	Candidates []Symbol    `json:"candidates,omitempty"`
	Ambiguous  bool        `json:"ambiguous"`
	Callers    []Edge      `json:"callers"`
	Callees    []Edge      `json:"callees"`
	Paths      [][]Edge    `json:"paths"`
	Upstream   [][]Edge    `json:"upstream"`
	Downstream [][]Edge    `json:"downstream"`
	Assessment *Assessment `json:"assessment,omitempty"`
	Truncated  bool        `json:"truncated"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	NextOffset *int        `json:"next_offset"`
	Omitted    []string    `json:"omitted"`
}

type symbol struct {
	id, path, name, short, kind, dir string
	line, end                        int
}

type rawEdge struct {
	fromID, toID       string
	fromPath, toPath   string
	fromName, toName   string
	fromLine, callLine int
	toLine             int
	kind, confidence   string
	callee             string
	reason             string
}

type index struct {
	root    string
	syms    []symbol
	byID    map[string]symbol
	edges   []rawEdge
	imports []pyImport
}

// Query traces callers, callees, or both (impact) for one symbol.
func Query(root string, cfg config.Config, query, mode string, limit, offset int) (Result, error) {
	mode = strings.TrimSpace(mode)
	switch mode {
	case "callers", "callees", "impact":
	default:
		return Result{}, fmt.Errorf("unknown trace mode %q", mode)
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if offset < 0 {
		offset = 0
	}
	ix, err := buildIndex(root, cfg)
	if err != nil {
		return Result{}, err
	}
	hits := ix.match(query)
	res := Result{
		Query:  query,
		Mode:   mode,
		Limit:  limit,
		Offset: offset,
		Omitted: []string{
			"calls that are not a unique name, import, route, dependency, callback, or registry stay unresolved",
			"impact paths are bounded to depth 6 and 8 paths",
		},
	}
	if len(hits) == 0 {
		return res, fmt.Errorf("no symbol %q", query)
	}
	if len(hits) != 1 {
		res.Ambiguous = true
		res.Assessment = &Assessment{
			Conclusion:  "ambiguous",
			StopReasons: []string{"ambiguous symbol: pass path:symbol from candidates"},
			Unsupported: []string{},
		}
		for _, h := range hits {
			res.Candidates = append(res.Candidates, exportSym(h))
		}
		sort.Slice(res.Candidates, func(i, j int) bool {
			if res.Candidates[i].Path != res.Candidates[j].Path {
				return res.Candidates[i].Path < res.Candidates[j].Path
			}
			return res.Candidates[i].Line < res.Candidates[j].Line
		})
		return res, nil
	}
	res.Resolved = exportSym(hits[0])
	if mode == "callers" || mode == "impact" {
		res.Callers = []Edge{}
	}
	if mode == "callees" || mode == "impact" {
		res.Callees = []Edge{}
	}
	var callers, callees []Edge
	for _, e := range ix.edges {
		if e.toID == hits[0].id || (e.toID == "" && nameMatch(e.callee, hits[0])) {
			callers = append(callers, Edge{
				Confidence: e.confidence, Kind: e.kind,
				File: e.fromPath, Line: e.callLine, Symbol: e.fromName,
				PeerFile: hits[0].path, PeerLine: hits[0].line, PeerSymbol: hits[0].name,
				CalleeText: e.callee, Reason: e.reason,
			})
		}
		if e.fromID == hits[0].id {
			callees = append(callees, Edge{
				Confidence: e.confidence, Kind: e.kind,
				File: e.fromPath, Line: e.callLine, Symbol: e.fromName,
				PeerFile: e.toPath, PeerLine: e.toLine, PeerSymbol: e.toName,
				CalleeText: e.callee, Reason: e.reason,
			})
		}
	}
	sortEdges(callers)
	sortEdges(callees)
	if mode == "callers" || mode == "impact" {
		res.Callers, res.Truncated = page(callers, limit, offset, res.Truncated)
	}
	if mode == "callees" || mode == "impact" {
		var cut bool
		res.Callees, cut = page(callees, limit, offset, false)
		res.Truncated = res.Truncated || cut
	}
	if res.Truncated {
		next := offset + limit
		res.NextOffset = &next
	}
	unresolved := 0
	for _, e := range append(append([]Edge{}, callers...), callees...) {
		if e.Confidence == "unresolved" {
			unresolved++
		}
	}
	direction := "both"
	switch mode {
	case "callers":
		direction = "upstream"
	case "callees":
		direction = "downstream"
	}
	assess := &Assessment{
		Direction:       direction,
		UnresolvedEdges: unresolved,
		StopReasons:     []string{},
		Unsupported:     []string{},
	}
	if mode == "impact" && !res.Ambiguous {
		const maxDepth, pathLimit = 6, 8
		up := ix.walkPaths(hits[0].id, true, maxDepth, pathLimit)
		down := ix.walkPaths(hits[0].id, false, maxDepth, pathLimit)
		res.Upstream = up.paths
		res.Downstream = down.paths
		res.Paths = up.paths
		if res.Upstream == nil {
			res.Upstream = [][]Edge{}
		}
		if res.Downstream == nil {
			res.Downstream = [][]Edge{}
		}
		if res.Paths == nil {
			res.Paths = [][]Edge{}
		}
		assess.MaxDepth = maxDepth
		assess.PathLimit = pathLimit
		assess.CycleCuts = up.cycles + down.cycles
		if up.depthCut || down.depthCut {
			assess.StopReasons = append(assess.StopReasons, "max depth")
		}
		if up.limited || down.limited {
			assess.StopReasons = append(assess.StopReasons, "path limit")
			res.Omitted = append(res.Omitted, "further impact paths omitted")
		}
		if assess.CycleCuts > 0 {
			assess.StopReasons = append(assess.StopReasons, "cycle")
		}
	}
	assess.Complete = unresolved == 0 && len(assess.StopReasons) == 0
	if assess.Complete {
		assess.Conclusion = "complete"
	} else {
		assess.Conclusion = "partial"
	}
	res.Assessment = assess
	return res, nil
}

type pathWalk struct {
	paths    [][]Edge
	depthCut bool
	limited  bool
	cycles   int
}

func (ix *index) walkPaths(start string, upstream bool, depth, maxPaths int) pathWalk {
	next := map[string][]rawEdge{}
	for _, e := range ix.edges {
		if upstream {
			if e.toID == "" {
				continue
			}
			next[e.toID] = append(next[e.toID], e)
		} else {
			if e.fromID == "" || e.toID == "" {
				continue
			}
			next[e.fromID] = append(next[e.fromID], e)
		}
	}
	return ix.outcomePaths(start, next, upstream, depth, maxPaths)
}

func (ix *index) outcomePaths(start string, next map[string][]rawEdge, upstream bool, depth, maxPaths int) pathWalk {
	type node struct {
		id    string
		chain []rawEdge
	}
	var out pathWalk
	q := []node{{id: start}}
	for len(q) > 0 && len(out.paths) < maxPaths {
		cur := q[0]
		q = q[1:]
		steps := next[cur.id]
		if len(steps) == 0 && len(cur.chain) > 0 {
			seq := append([]rawEdge{}, cur.chain...)
			if upstream {
				for i, j := 0, len(seq)-1; i < j; i, j = i+1, j-1 {
					seq[i], seq[j] = seq[j], seq[i]
				}
			}
			var path []Edge
			for _, e := range seq {
				path = append(path, Edge{
					Confidence: e.confidence, Kind: e.kind, File: e.fromPath, Line: e.callLine,
					Symbol: e.fromName, PeerFile: e.toPath, PeerLine: e.toLine, PeerSymbol: e.toName,
					CalleeText: e.callee, Reason: e.reason,
				})
			}
			out.paths = append(out.paths, path)
			continue
		}
		if len(cur.chain) >= depth {
			if len(steps) > 0 {
				out.depthCut = true
			}
			continue
		}
		for _, e := range steps {
			hop := e.fromID
			if !upstream {
				hop = e.toID
			}
			if hop == "" || hop == start {
				continue
			}
			cycle := false
			for _, prev := range cur.chain {
				prevHop := prev.fromID
				if !upstream {
					prevHop = prev.toID
				}
				if prevHop == hop {
					cycle = true
					break
				}
			}
			if cycle {
				out.cycles++
				continue
			}
			chain := append(append([]rawEdge{}, cur.chain...), e)
			q = append(q, node{id: hop, chain: chain})
		}
	}
	if len(q) > 0 {
		out.limited = true
	}
	if out.paths == nil {
		out.paths = [][]Edge{}
	}
	return out
}

func page(in []Edge, limit, offset int, already bool) ([]Edge, bool) {
	if offset >= len(in) {
		return []Edge{}, already
	}
	end := offset + limit
	trunc := already
	if end < len(in) {
		trunc = true
	} else {
		end = len(in)
	}
	out := append([]Edge(nil), in[offset:end]...)
	return out, trunc
}

func sortEdges(es []Edge) {
	sort.Slice(es, func(i, j int) bool {
		if es[i].File != es[j].File {
			return es[i].File < es[j].File
		}
		if es[i].Line != es[j].Line {
			return es[i].Line < es[j].Line
		}
		return es[i].Symbol < es[j].Symbol
	})
}

func nameMatch(callee string, s symbol) bool {
	callee = strings.TrimSpace(callee)
	if i := strings.LastIndex(callee, "."); i >= 0 {
		callee = callee[i+1:]
	}
	return callee == s.short || callee == s.name
}

func exportSym(s symbol) Symbol {
	return Symbol{ID: s.id, Path: s.path, Name: s.name, Short: s.short, Kind: s.kind, Line: s.line}
}

func (ix *index) match(q string) []symbol {
	q = strings.TrimSpace(q)
	pathHint, name := "", q
	if i := strings.LastIndex(q, ":"); i >= 0 {
		pathHint = strings.TrimPrefix(filepath.ToSlash(q[:i]), "./")
		name = q[i+1:]
	}
	var hits []symbol
	for _, s := range ix.syms {
		if pathHint != "" && s.path != pathHint && !strings.HasSuffix(s.path, "/"+pathHint) && !strings.Contains(s.path, pathHint) {
			continue
		}
		if strings.Contains(name, ".") {
			if s.name == name {
				hits = append(hits, s)
			}
			continue
		}
		if s.short == name || s.name == name {
			hits = append(hits, s)
		}
	}
	return hits
}

func (ix *index) addSym(path, name, kind string, line int) symbol {
	short := name
	if i := strings.LastIndex(name, "."); i >= 0 {
		short = name[i+1:]
	}
	s := symbol{
		id:   path + "#" + name + "#" + fmt.Sprint(line),
		path: path, name: name, short: short, kind: kind, line: line,
		dir: filepath.ToSlash(filepath.Dir(path)),
		end: line,
	}
	ix.syms = append(ix.syms, s)
	ix.byID[s.id] = s
	return s
}

func buildIndex(root string, cfg config.Config) (*index, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	ix := &index{root: rootAbs, byID: map[string]symbol{}}
	if err != nil {
		return nil, err
	}
	_, err = measure.NewScope(rootAbs, cfg).Walk(rootAbs, -1, func(path, rel string) error {
		ext := strings.ToLower(filepath.Ext(rel))
		switch ext {
		case ".go":
			if !goFileWanted(path) || generatedGo(path) {
				return nil
			}
			ix.indexGo(path, rel)
		case ".py":
			ix.indexPy(rel, path)
		case ".ts", ".tsx", ".js", ".jsx":
			ix.indexTS(rel, path)
		}
		return nil
	})
	ix.resolveGoDirect()
	ix.resolveGoTypes()
	ix.resolveGoImports()
	ix.resolvePyMethods()
	ix.resolvePyImports()
	ix.resolveUniqueNames()
	ix.resolveFrameworkBinds()
	ix.dedupeEdges()
	return ix, err
}

func goFileWanted(path string) bool {
	dir := filepath.Dir(path)
	ctx := build.Default
	ctx.Dir = dir
	ok, err := ctx.MatchFile(dir, filepath.Base(path))
	if err != nil {
		return true
	}
	return ok
}

func generatedGo(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 2048)
	n, _ := f.Read(buf)
	head := string(buf[:n])
	return strings.Contains(head, "Code generated") && strings.Contains(head, "DO NOT EDIT")
}

func (ix *index) indexGo(abs, rel string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, abs, nil, 0)
	if err != nil {
		return
	}
	dir := filepath.ToSlash(filepath.Dir(rel))
	var fns []*ast.FuncDecl
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		fns = append(fns, fn)
		name := fn.Name.Name
		kind := "func"
		if recv := recvType(fn); recv != "" {
			name = recv + "." + name
			kind = "method"
		}
		ix.addSym(rel, name, kind, fset.Position(fn.Pos()).Line)
	}
	for _, fn := range fns {
		name := fn.Name.Name
		if recv := recvType(fn); recv != "" {
			name = recv + "." + name
		}
		s := ix.lookupNamed(rel, name)
		recvName, recvT := recvIdent(fn)
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			line := fset.Position(call.Pos()).Line
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				ix.addGoIdentCall(dir, s, line, fun.Name)
			case *ast.SelectorExpr:
				sel := fun.Sel.Name
				if id, ok := fun.X.(*ast.Ident); ok && recvName != "" && id.Name == recvName && ix.methodExists(dir, recvT, sel) {
					target := ix.lookupMethod(dir, recvT, sel)
					ix.edges = append(ix.edges, rawEdge{
						fromID: s.id, toID: target.id,
						fromPath: rel, toPath: target.path,
						fromName: s.name, toName: target.name,
						fromLine: s.line, callLine: line, toLine: target.line,
						kind: "method", confidence: "statically-confirmed",
						callee: recvT + "." + sel,
					})
					return true
				}
				text := sel
				if id, ok := fun.X.(*ast.Ident); ok {
					text = id.Name + "." + sel
				}
				ix.edges = append(ix.edges, rawEdge{
					fromID: s.id, fromPath: rel, fromName: s.name, fromLine: s.line,
					callLine: line, kind: "selector", confidence: "unresolved",
					callee: text,
				})
			}
			return true
		})
	}
}

func (ix *index) addGoIdentCall(dir string, from symbol, line int, name string) {
	ix.edges = append(ix.edges, rawEdge{
		fromID: from.id, fromPath: from.path, fromName: from.name, fromLine: from.line,
		callLine: line, callee: name, kind: "direct", confidence: "unresolved",
	})
	_ = dir
}

func (ix *index) resolveGoDirect() {
	for i := range ix.edges {
		e := &ix.edges[i]
		if e.kind != "direct" || e.toID != "" || e.confidence != "unresolved" {
			continue
		}
		from, ok := ix.byID[e.fromID]
		if !ok {
			continue
		}
		matches := ix.funcsInDir(from.dir, e.callee)
		if len(matches) != 1 {
			continue
		}
		e.toID = matches[0].id
		e.toPath = matches[0].path
		e.toName = matches[0].name
		e.toLine = matches[0].line
		e.confidence = "statically-confirmed"
	}
}

func (ix *index) lookupNamed(path, name string) symbol {
	for _, s := range ix.syms {
		if s.path == path && s.name == name {
			return s
		}
	}
	return symbol{}
}

func (ix *index) funcsInDir(dir, name string) []symbol {
	var out []symbol
	for _, s := range ix.syms {
		if s.dir == dir && s.kind == "func" && s.name == name {
			out = append(out, s)
		}
	}
	return out
}

func (ix *index) methodExists(dir, recv, name string) bool {
	return ix.lookupMethod(dir, recv, name).id != ""
}

func (ix *index) lookupMethod(dir, recv, name string) symbol {
	qual := recv + "." + name
	for _, s := range ix.syms {
		if s.dir == dir && s.kind == "method" && s.name == qual {
			return s
		}
	}
	return symbol{}
}

func recvType(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	return exprName(fn.Recv.List[0].Type)
}

func recvIdent(fn *ast.FuncDecl) (string, string) {
	if fn.Recv == nil || len(fn.Recv.List) == 0 || len(fn.Recv.List[0].Names) == 0 {
		return "", recvType(fn)
	}
	return fn.Recv.List[0].Names[0].Name, recvType(fn)
}

func exprName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return exprName(t.X)
	case *ast.IndexExpr:
		return exprName(t.X)
	default:
		return ""
	}
}

var (
	rePyClass      = regexp.MustCompile(`^class\s+([A-Za-z_][A-Za-z0-9_]*)`)
	rePyDef        = regexp.MustCompile(`^(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)`)
	reTSFunc       = regexp.MustCompile(`^(?:export\s+(?:default\s+)?)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	reTSConst      = regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*(?:async\s*)?(?:\(|[A-Za-z_$])`)
	reTSConstStart = regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=`)
	reCall         = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	reSelCall      = regexp.MustCompile(`\b(self|cls)\.([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
)

func (ix *index) findInFile(rel, name string) symbol {
	for _, s := range ix.syms {
		if s.path == rel && s.name == name {
			return s
		}
	}
	return symbol{}
}

func (ix *index) uniqueInFile(rel, short string) (symbol, int) {
	var hit symbol
	n := 0
	for _, s := range ix.syms {
		if s.path == rel && s.kind != "class" && s.short == short {
			hit = s
			n++
		}
	}
	return hit, n
}

func indentOf(line string) int {
	n := 0
	for _, r := range line {
		if r == ' ' {
			n++
			continue
		}
		if r == '\t' {
			n += 4
			continue
		}
		break
	}
	return n
}

func keyword(name string) bool {
	switch name {
	case "if", "for", "while", "with", "return", "def", "class", "except", "elif", "switch", "catch", "function", "super", "print":
		return true
	default:
		return false
	}
}

// FormatText renders a trace for a terminal.
func FormatText(res Result) string {
	var b strings.Builder
	if res.Ambiguous {
		fmt.Fprintf(&b, "ambiguous %s\n", res.Query)
		for _, c := range res.Candidates {
			fmt.Fprintf(&b, "  %s:%d %s\n", c.Path, c.Line, c.Name)
		}
		return b.String()
	}
	if res.Resolved.Name == "" {
		fmt.Fprintf(&b, "no symbol %s\n", res.Query)
		return b.String()
	}
	fmt.Fprintf(&b, "%s %s:%d %s\n", res.Resolved.Kind, res.Resolved.Path, res.Resolved.Line, res.Resolved.Name)
	write := func(title string, es []Edge) {
		fmt.Fprintf(&b, "%s %d\n", title, len(es))
		for _, e := range es {
			peer := e.PeerSymbol
			if peer == "" {
				peer = e.CalleeText
			}
			fmt.Fprintf(&b, "  %s %s %s:%d %s -> %s\n", e.Confidence, e.Kind, e.File, e.Line, e.Symbol, peer)
		}
	}
	if res.Mode != "callees" {
		write("callers", res.Callers)
	}
	if res.Mode != "callers" {
		write("callees", res.Callees)
	}
	if res.Truncated {
		next := 0
		if res.NextOffset != nil {
			next = *res.NextOffset
		}
		fmt.Fprintf(&b, "truncated: true next_offset: %d\n", next)
	}
	return b.String()
}
