// Package gocyclo counts the cyclomatic number of Go functions. cyclotree
// and the c3po ast patch gate both use it, so they report one number.
package gocyclo

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// Name is the name cyclotree gives a function declaration: the receiver
// type without its pointer star or type parameters, a dot, then the name.
func Name(fn *ast.FuncDecl) string {
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

// File scores every function declaration in src by Name. It reports false
// when go/parser rejects src or two declarations share a name.
func File(src []byte) (map[string]int, bool) {
	f, err := parser.ParseFile(token.NewFileSet(), "", src, 0)
	if err != nil || f == nil {
		return nil, false
	}
	out := map[string]int{}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := Name(fn)
		if _, dup := out[name]; dup {
			return nil, false
		}
		out[name] = Body(fn.Body)
	}
	return out, true
}

// Body returns the cyclomatic number of one function body. It starts at 1.
// if, else if, for, and range add 1. A switch adds 1 per case, or 1 when it
// has none. A select adds 1 and 1 per comm clause. Each && and || adds 1.
// A function literal inside the body adds nothing.
func Body(body *ast.BlockStmt) int {
	c := &counter{n: 1}
	if body != nil {
		c.stmt(body)
	}
	return c.n
}

type counter struct{ n int }

func (c *counter) stmt(n ast.Node) {
	switch x := n.(type) {
	case nil:
	case *ast.BlockStmt:
		c.list(x.List)
	case *ast.IfStmt:
		c.stmt(x.Init)
		c.n++
		c.cond(x.Cond)
		c.stmt(x.Body)
		c.els(x.Else)
	case *ast.ForStmt:
		c.stmt(x.Init)
		c.n++
		c.cond(x.Cond)
		c.stmt(x.Post)
		c.stmt(x.Body)
	case *ast.RangeStmt:
		c.n++
		c.expr(x.X)
		c.stmt(x.Body)
	default:
		c.branch(n)
	}
}

func (c *counter) branch(n ast.Node) {
	switch x := n.(type) {
	case *ast.SwitchStmt:
		c.stmt(x.Init)
		c.expr(x.Tag)
		c.cases(x.Body)
	case *ast.TypeSwitchStmt:
		c.stmt(x.Init)
		c.cases(x.Body)
	case *ast.SelectStmt:
		c.sel(x)
	default:
		c.simple(n)
	}
}

func (c *counter) simple(n ast.Node) {
	switch x := n.(type) {
	case *ast.ExprStmt:
		c.expr(x.X)
	case *ast.ReturnStmt:
		c.exprs(x.Results)
	case *ast.AssignStmt:
		c.exprs(x.Rhs)
	case *ast.GoStmt:
		c.expr(x.Call)
	case *ast.DeferStmt:
		c.expr(x.Call)
	default:
		c.simpleMore(n)
	}
}

func (c *counter) simpleMore(n ast.Node) {
	switch x := n.(type) {
	case *ast.DeclStmt:
		c.decl(x.Decl)
	case *ast.IncDecStmt:
		c.expr(x.X)
	case *ast.SendStmt:
		c.expr(x.Chan)
		c.expr(x.Value)
	case *ast.LabeledStmt:
		c.stmt(x.Stmt)
	}
}

func (c *counter) list(list []ast.Stmt) {
	for _, s := range list {
		c.stmt(s)
	}
}

func (c *counter) els(els ast.Stmt) {
	elif, ok := els.(*ast.IfStmt)
	if !ok {
		if els != nil {
			c.stmt(els)
		}
		return
	}
	c.n++
	c.stmt(elif.Init)
	c.cond(elif.Cond)
	c.stmt(elif.Body)
	c.els(elif.Else)
}

func (c *counter) cases(body *ast.BlockStmt) {
	cases := 0
	if body != nil {
		for _, s := range body.List {
			cc, ok := s.(*ast.CaseClause)
			if !ok {
				continue
			}
			cases++
			c.n++
			c.list(cc.Body)
		}
	}
	if cases == 0 {
		c.n++
	}
}

func (c *counter) sel(x *ast.SelectStmt) {
	c.n++
	if x.Body == nil {
		return
	}
	for _, s := range x.Body.List {
		cc, ok := s.(*ast.CommClause)
		if !ok {
			continue
		}
		c.n++
		c.list(cc.Body)
	}
}

func (c *counter) decl(d ast.Decl) {
	gen, ok := d.(*ast.GenDecl)
	if !ok {
		return
	}
	for _, spec := range gen.Specs {
		if vs, ok := spec.(*ast.ValueSpec); ok {
			c.exprs(vs.Values)
		}
	}
}

func (c *counter) exprs(list []ast.Expr) {
	for _, e := range list {
		c.expr(e)
	}
}

func (c *counter) expr(e ast.Expr) {
	switch x := e.(type) {
	case nil, *ast.FuncLit:
	case *ast.BinaryExpr:
		if x.Op == token.LAND || x.Op == token.LOR {
			c.cond(x)
			return
		}
		c.expr(x.X)
		c.expr(x.Y)
	case *ast.ParenExpr:
		c.expr(x.X)
	case *ast.CallExpr:
		c.expr(x.Fun)
		c.exprs(x.Args)
	default:
		c.exprMore(e)
	}
}

func (c *counter) exprMore(e ast.Expr) {
	switch x := e.(type) {
	case *ast.SelectorExpr:
		c.expr(x.X)
	case *ast.StarExpr:
		c.expr(x.X)
	case *ast.UnaryExpr:
		c.expr(x.X)
	case *ast.IndexExpr:
		c.expr(x.X)
		c.expr(x.Index)
	case *ast.SliceExpr:
		c.expr(x.X)
		c.expr(x.Low)
		c.expr(x.High)
		c.expr(x.Max)
	case *ast.CompositeLit:
		c.exprs(x.Elts)
	case *ast.KeyValueExpr:
		c.expr(x.Key)
		c.expr(x.Value)
	case *ast.TypeAssertExpr:
		c.expr(x.X)
	}
}

// cond adds 1 for each && and || in one boolean chain and walks the
// operands that are not part of the chain.
func (c *counter) cond(e ast.Expr) {
	c.n += c.ops(e)
}

func (c *counter) ops(e ast.Expr) int {
	b, ok := e.(*ast.BinaryExpr)
	if !ok || (b.Op != token.LAND && b.Op != token.LOR) {
		c.expr(e)
		return 0
	}
	return c.ops(b.X) + 1 + c.ops(b.Y)
}
