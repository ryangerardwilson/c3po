package trace

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// SymbolSpan returns the inclusive line range for a symbol name in one file.
func SymbolSpan(abs, rel, name string) (int, int, bool) {
	if strings.HasSuffix(strings.ToLower(rel), ".go") {
		return goSpan(abs, name)
	}
	for _, o := range FileOutline(abs, rel) {
		if o.Name == name || strings.HasSuffix(o.Name, "."+name) {
			end := o.End
			if end < o.Line {
				end = o.Line
			}
			return o.Line, end, true
		}
	}
	return 0, 0, false
}

func goSpan(abs, name string) (int, int, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, abs, nil, 0)
	if err != nil {
		return 0, 0, false
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		qual := fn.Name.Name
		if recv := recvType(fn); recv != "" {
			qual = recv + "." + qual
		}
		if qual == name || fn.Name.Name == name {
			return fset.Position(fn.Pos()).Line, fset.Position(fn.End()).Line, true
		}
	}
	_ = filepath.Separator
	return 0, 0, false
}
