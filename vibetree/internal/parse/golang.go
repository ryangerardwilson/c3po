package parse

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"hash/fnv"
	"strconv"

	"github.com/ryangerardwilson/c3po/vibetree/internal/metric"
)

// ParseGo scores every function and function literal in src.
func ParseGo(file string, src []byte) ([]metric.Function, []string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	var diags []string
	if err != nil {
		diags = append(diags, err.Error())
	}
	if f == nil {
		return nil, diags
	}
	code := CodeLines(string(src), "go")
	var out []metric.Function
	perLine := map[int]int{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			out = append(out, scoreGo(goFn{fset, code, file, declName(fn), fn.Type, fn.Body, fn.Pos(), fn.End()}))
		case *ast.FuncLit:
			out = append(out, scoreLit(goFn{fset, code, file, "", fn.Type, fn.Body, fn.Pos(), fn.End()}, src, perLine))
		}
		return true
	})
	return out, diags
}

// scoreLit names a function literal by its line. A second literal on the
// same line gets a #2 suffix. Sig is what matches it across edits.
func scoreLit(fn goFn, src []byte, perLine map[int]int) metric.Function {
	line := fn.fset.Position(fn.pos).Line
	perLine[line]++
	fn.name = fmt.Sprintf("func@%d", line)
	if n := perLine[line]; n > 1 {
		fn.name = fmt.Sprintf("%s#%d", fn.name, n)
	}
	out := scoreGo(fn)
	out.Sig = goSig(src, fn.fset, fn.pos, fn.end)
	return out
}

// goSig hashes the tokens of one function literal. Comments and layout do
// not change it.
func goSig(src []byte, fset *token.FileSet, pos, end token.Pos) string {
	tf := fset.File(pos)
	if tf == nil {
		return ""
	}
	lo, hi := tf.Offset(pos), tf.Offset(end)
	if lo < 0 || hi > len(src) || lo >= hi {
		return ""
	}
	var sc scanner.Scanner
	sc.Init(token.NewFileSet().AddFile("", -1, hi-lo), src[lo:hi], nil, 0)
	h := fnv.New64a()
	for {
		_, tok, lit := sc.Scan()
		if tok == token.EOF {
			break
		}
		_, _ = h.Write([]byte(tok.String() + lit + "\x00"))
	}
	return strconv.FormatUint(h.Sum64(), 16)
}

func declName(fn *ast.FuncDecl) string {
	name := "func"
	if fn.Name != nil {
		name = fn.Name.Name
	}
	recv := recvName(fn)
	if recv == "" {
		return name
	}
	return recv + "." + name
}

func recvName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	return typeName(fn.Recv.List[0].Type)
}

func typeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return typeName(t.X)
	case *ast.IndexExpr:
		return typeName(t.X)
	case *ast.IndexListExpr:
		return typeName(t.X)
	default:
		return ""
	}
}

type goFn struct {
	fset *token.FileSet
	code []bool
	file string
	name string
	typ  *ast.FuncType
	body *ast.BlockStmt
	pos  token.Pos
	end  token.Pos
}

func scoreGo(fn goFn) metric.Function {
	start := fn.fset.Position(fn.pos).Line
	stop := fn.fset.Position(fn.end).Line
	st := &gstate{fset: fn.fset, name: shortName(fn.name), cyclo: 1}
	if fn.body != nil {
		walkStmt(fn.body, 0, st)
	}
	if st.recur.IsValid() {
		st.add(st.recur, metric.Event{Kind: "recursion", Cog: 1})
	}
	return metric.Function{
		File: fn.file, Line: start, EndLine: stop, Name: fn.name, Language: "go",
		Cognitive: st.cog, Nesting: st.nest, Lines: countBetween(fn.code, start, stop),
		Params: paramCount(fn.typ), Cyclomatic: st.cyclo, Events: st.events,
	}
}

func shortName(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i+1:]
		}
	}
	return name
}

func paramCount(typ *ast.FuncType) int {
	if typ == nil || typ.Params == nil {
		return 0
	}
	n := 0
	for _, field := range typ.Params.List {
		if len(field.Names) == 0 {
			n++
			continue
		}
		n += len(field.Names)
	}
	return n
}

type gstate struct {
	fset   *token.FileSet
	name   string
	cog    int
	nest   int
	cyclo  int
	recur  token.Pos
	events []metric.Event
}

func (st *gstate) add(pos token.Pos, e metric.Event) {
	e.Line = st.fset.Position(pos).Line
	st.cog += e.Cog
	st.cyclo += e.Cyclo
	if e.Depth > st.nest {
		st.nest = e.Depth
	}
	st.events = append(st.events, e)
}

func (st *gstate) control(pos token.Pos, kind string, nesting int) {
	st.add(pos, metric.Event{Kind: kind, Nesting: nesting, Cog: 1 + nesting, Cyclo: 1, Depth: nesting + 1})
}

func walkStmt(n ast.Node, nesting int, st *gstate) {
	switch x := n.(type) {
	case nil:
	case *ast.BlockStmt:
		walkList(x.List, nesting, st)
	case *ast.IfStmt:
		walkIf(x, nesting, st)
	case *ast.ForStmt:
		walkFor(x, nesting, st)
	case *ast.RangeStmt:
		walkRange(x, nesting, st)
	default:
		walkStmtMore(n, nesting, st)
	}
}

func walkStmtMore(n ast.Node, nesting int, st *gstate) {
	switch x := n.(type) {
	case *ast.SwitchStmt:
		walkSwitch(x.Body, x.Init, x.Tag, nesting, st)
	case *ast.TypeSwitchStmt:
		walkSwitch(x.Body, x.Init, nil, nesting, st)
	case *ast.SelectStmt:
		walkSelect(x, nesting, st)
	case *ast.BranchStmt:
		if x.Label != nil {
			st.add(x.Pos(), metric.Event{Kind: x.Tok.String() + " label", Nesting: nesting, Cog: 1})
		}
	default:
		walkOther(n, nesting, st)
	}
}

func walkList(list []ast.Stmt, nesting int, st *gstate) {
	for _, s := range list {
		walkStmt(s, nesting, st)
	}
}

func walkIf(x *ast.IfStmt, nesting int, st *gstate) {
	walkStmt(x.Init, nesting, st)
	st.control(x.If, "if", nesting)
	addBool(x.Cond, nesting, st)
	walkStmt(x.Body, nesting+1, st)
	walkElse(x.Else, nesting, st)
}

func walkElse(els ast.Stmt, nesting int, st *gstate) {
	elif, ok := els.(*ast.IfStmt)
	if !ok {
		if els != nil {
			st.add(els.Pos(), metric.Event{Kind: "else", Nesting: nesting, Cog: 1})
			walkStmt(els, nesting+1, st)
		}
		return
	}
	st.add(elif.If, metric.Event{Kind: "else if", Nesting: nesting, Cog: 1, Cyclo: 1})
	walkStmt(elif.Init, nesting, st)
	addBool(elif.Cond, nesting, st)
	walkStmt(elif.Body, nesting+1, st)
	walkElse(elif.Else, nesting, st)
}

func walkFor(x *ast.ForStmt, nesting int, st *gstate) {
	walkStmt(x.Init, nesting, st)
	st.control(x.For, "for", nesting)
	addBool(x.Cond, nesting, st)
	walkStmt(x.Post, nesting, st)
	walkStmt(x.Body, nesting+1, st)
}

func walkRange(x *ast.RangeStmt, nesting int, st *gstate) {
	st.control(x.For, "for", nesting)
	walkExpr(x.X, nesting, st)
	walkStmt(x.Body, nesting+1, st)
}

func walkSwitch(body *ast.BlockStmt, init ast.Stmt, tag ast.Expr, nesting int, st *gstate) {
	walkStmt(init, nesting, st)
	walkExpr(tag, nesting, st)
	pos := token.NoPos
	var list []ast.Stmt
	if body != nil {
		pos, list = body.Lbrace, body.List
	}
	st.add(pos, metric.Event{Kind: "switch", Nesting: nesting, Cog: 1 + nesting, Depth: nesting + 1})
	cases := 0
	for _, s := range list {
		cc, ok := s.(*ast.CaseClause)
		if !ok {
			continue
		}
		cases++
		st.add(cc.Case, metric.Event{Kind: "case", Nesting: nesting + 1, Cyclo: 1})
		walkList(cc.Body, nesting+1, st)
	}
	if cases == 0 {
		st.cyclo++
	}
}

func walkSelect(x *ast.SelectStmt, nesting int, st *gstate) {
	st.control(x.Select, "select", nesting)
	if x.Body == nil {
		return
	}
	for _, s := range x.Body.List {
		cc, ok := s.(*ast.CommClause)
		if !ok {
			continue
		}
		st.add(cc.Case, metric.Event{Kind: "case", Nesting: nesting + 1, Cyclo: 1})
		walkList(cc.Body, nesting+1, st)
	}
}

func walkOther(n ast.Node, nesting int, st *gstate) {
	switch x := n.(type) {
	case *ast.ExprStmt:
		walkExpr(x.X, nesting, st)
	case *ast.ReturnStmt:
		walkExprs(x.Results, nesting, st)
	case *ast.AssignStmt:
		walkExprs(x.Rhs, nesting, st)
	case *ast.GoStmt:
		walkExpr(x.Call, nesting, st)
	case *ast.DeferStmt:
		walkExpr(x.Call, nesting, st)
	default:
		walkOtherMore(n, nesting, st)
	}
}

func walkOtherMore(n ast.Node, nesting int, st *gstate) {
	switch x := n.(type) {
	case *ast.DeclStmt:
		walkDecl(x.Decl, nesting, st)
	case *ast.IncDecStmt:
		walkExpr(x.X, nesting, st)
	case *ast.SendStmt:
		walkExpr(x.Chan, nesting, st)
		walkExpr(x.Value, nesting, st)
	case *ast.LabeledStmt:
		walkStmt(x.Stmt, nesting, st)
	}
}

func walkDecl(d ast.Decl, nesting int, st *gstate) {
	gen, ok := d.(*ast.GenDecl)
	if !ok {
		return
	}
	for _, spec := range gen.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		walkExprs(vs.Values, nesting, st)
	}
}

func walkExprs(list []ast.Expr, nesting int, st *gstate) {
	for _, e := range list {
		walkExpr(e, nesting, st)
	}
}

func walkExpr(e ast.Expr, nesting int, st *gstate) {
	switch x := e.(type) {
	case nil:
	case *ast.FuncLit:
		st.add(x.Pos(), metric.Event{Kind: "func literal", Nesting: nesting, Cog: 1 + nesting})
	case *ast.BinaryExpr:
		walkBinary(x, nesting, st)
	case *ast.ParenExpr:
		walkExpr(x.X, nesting, st)
	case *ast.CallExpr:
		noteCall(x, st)
		walkExpr(x.Fun, nesting, st)
		walkExprs(x.Args, nesting, st)
	default:
		walkExprMore(e, nesting, st)
	}
}

func walkBinary(x *ast.BinaryExpr, nesting int, st *gstate) {
	if x.Op == token.LAND || x.Op == token.LOR {
		addBool(x, nesting, st)
		return
	}
	walkExpr(x.X, nesting, st)
	walkExpr(x.Y, nesting, st)
}

func walkExprMore(e ast.Expr, nesting int, st *gstate) {
	switch x := e.(type) {
	case *ast.SelectorExpr:
		walkExpr(x.X, nesting, st)
	case *ast.StarExpr:
		walkExpr(x.X, nesting, st)
	case *ast.UnaryExpr:
		walkExpr(x.X, nesting, st)
	case *ast.IndexExpr:
		walkExpr(x.X, nesting, st)
		walkExpr(x.Index, nesting, st)
	case *ast.SliceExpr:
		walkExpr(x.X, nesting, st)
		walkExpr(x.Low, nesting, st)
		walkExpr(x.High, nesting, st)
		walkExpr(x.Max, nesting, st)
	case *ast.CompositeLit:
		for _, elt := range x.Elts {
			walkLitElt(elt, nesting, st)
		}
	case *ast.KeyValueExpr:
		walkExpr(x.Key, nesting, st)
		walkExpr(x.Value, nesting, st)
	case *ast.TypeAssertExpr:
		walkExpr(x.X, nesting, st)
	}
}

func walkLitElt(e ast.Expr, nesting int, st *gstate) {
	if kv, ok := e.(*ast.KeyValueExpr); ok {
		walkExpr(kv.Key, nesting, st)
		walkExpr(kv.Value, nesting, st)
		return
	}
	walkExpr(e, nesting, st)
}

func noteCall(c *ast.CallExpr, st *gstate) {
	if !st.recur.IsValid() && calleeName(c.Fun) == st.name {
		st.recur = c.Pos()
	}
}

func calleeName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		if f.Sel != nil {
			return f.Sel.Name
		}
	}
	return ""
}

func addBool(e ast.Expr, nesting int, st *gstate) {
	ops := flattenBool(e, nesting, st)
	if len(ops) == 0 {
		return
	}
	cog := 1
	for i := 1; i < len(ops); i++ {
		if ops[i] != ops[i-1] {
			cog++
		}
	}
	st.add(e.Pos(), metric.Event{Kind: ops[0].String(), Nesting: nesting, Cog: cog, Cyclo: len(ops)})
}

func flattenBool(e ast.Expr, nesting int, st *gstate) []token.Token {
	b, ok := e.(*ast.BinaryExpr)
	if !ok || (b.Op != token.LAND && b.Op != token.LOR) {
		walkExpr(e, nesting, st)
		return nil
	}
	left := flattenBool(b.X, nesting, st)
	right := flattenBool(b.Y, nesting, st)
	return append(append(left, b.Op), right...)
}
