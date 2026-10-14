package trace

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

func (ix *index) resolveGoTypes() {
	byDir := map[string][]string{}
	seen := map[string]bool{}
	for _, s := range ix.syms {
		if !strings.HasSuffix(s.path, ".go") || seen[s.path] {
			continue
		}
		seen[s.path] = true
		byDir[s.dir] = append(byDir[s.dir], s.path)
	}
	for dir, rels := range byDir {
		fields := map[string]string{}
		ifaces := map[string]map[string]bool{}
		concretes := map[string][]string{}
		for _, rel := range rels {
			ix.collectGoTypes(rel, fields, ifaces, concretes)
		}
		for _, rel := range rels {
			ix.resolveGoFile(dir, rel, fields, ifaces, concretes)
		}
	}
}

func (ix *index) collectGoTypes(rel string, fields map[string]string, ifaces map[string]map[string]bool, concretes map[string][]string) {
	abs := filepath.Join(ix.root, filepath.FromSlash(rel))
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, abs, nil, 0)
	if err != nil {
		return
	}
	dir := filepath.ToSlash(filepath.Dir(rel))
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			switch t := ts.Type.(type) {
			case *ast.StructType:
				if t.Fields == nil {
					continue
				}
				for _, field := range t.Fields.List {
					tn := typeString(field.Type)
					for _, name := range field.Names {
						fields[dir+"|"+ts.Name.Name+"|"+name.Name] = tn
					}
				}
			case *ast.InterfaceType:
				set := map[string]bool{}
				if t.Methods != nil {
					for _, field := range t.Methods.List {
						for _, name := range field.Names {
							set[name.Name] = true
							ix.addSym(rel, ts.Name.Name+"."+name.Name, "interface-method", fset.Position(name.Pos()).Line)
						}
					}
				}
				ifaces[dir+"|"+ts.Name.Name] = set
			}
		}
	}
	for _, s := range ix.syms {
		if s.dir == dir && s.kind == "method" {
			if i := strings.Index(s.name, "."); i > 0 {
				concretes[s.name[i+1:]] = append(concretes[s.name[i+1:]], s.name[:i])
			}
		}
	}
}

func (ix *index) resolveGoFile(dir, rel string, fields map[string]string, ifaces map[string]map[string]bool, concretes map[string][]string) {
	abs := filepath.Join(ix.root, filepath.FromSlash(rel))
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, abs, nil, 0)
	if err != nil {
		return
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Name == nil {
			continue
		}
		name := fn.Name.Name
		recvName, recvT := recvIdent(fn)
		if recvT != "" {
			name = recvT + "." + name
		}
		from := ix.lookupNamed(rel, name)
		if from.id == "" {
			continue
		}
		locals := map[string]string{}
		valueBinds := map[string]string{}
		if recvName != "" {
			locals[recvName] = recvT
		}
		if fn.Type.Params != nil {
			for _, field := range fn.Type.Params.List {
				tn := typeString(field.Type)
				for _, n := range field.Names {
					locals[n.Name] = strings.TrimPrefix(tn, "*")
				}
			}
		}
		pkgVars := map[string]string{}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok.String() != "var" {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || vs.Type == nil {
					continue
				}
				tn := strings.TrimPrefix(typeString(vs.Type), "*")
				for _, name := range vs.Names {
					pkgVars[name.Name] = tn
				}
			}
		}
		lookupBound := func(stored string) symbol {
			target := ix.lookupNamed(rel, stored)
			if target.id == "" {
				if dot := strings.LastIndex(stored, "."); dot > 0 {
					target = ix.lookupMethod(dir, stored[:dot], stored[dot+1:])
				}
			}
			return target
		}
		resolveValue := func(e ast.Expr) string {
			switch t := e.(type) {
			case *ast.Ident:
				if b := valueBinds[t.Name]; b != "" {
					return b
				}
				return ""
			case *ast.SelectorExpr:
				id, ok := t.X.(*ast.Ident)
				if !ok {
					return ""
				}
				if b := valueBinds[id.Name+"."+t.Sel.Name]; b != "" {
					return b
				}
				owner := locals[id.Name]
				if owner == "" {
					owner = pkgVars[id.Name]
				}
				owner = strings.TrimPrefix(owner, "*")
				if owner == "" {
					return ""
				}
				if stored := fields[dir+"|"+owner+"|"+t.Sel.Name+"=stored"]; stored != "" {
					return stored
				}
				if ix.methodExists(dir, owner, t.Sel.Name) {
					return owner + "." + t.Sel.Name
				}
			}
			return ""
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			as, ok := n.(*ast.AssignStmt)
			if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
				return true
			}
			if id, ok := as.Lhs[0].(*ast.Ident); ok {
				if cl, ok := as.Rhs[0].(*ast.CompositeLit); ok {
					if tn := strings.TrimPrefix(typeString(cl.Type), "*"); tn != "" {
						locals[id.Name] = tn
					}
					for _, elt := range cl.Elts {
						kv, ok := elt.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						key, ok := kv.Key.(*ast.Ident)
						if !ok {
							continue
						}
						if bound := resolveValue(kv.Value); bound != "" {
							valueBinds[id.Name+"."+key.Name] = bound
						}
					}
				}
				if bound := resolveValue(as.Rhs[0]); bound != "" {
					valueBinds[id.Name] = bound
				}
				return true
			}
			lhs, ok := as.Lhs[0].(*ast.SelectorExpr)
			if !ok {
				return true
			}
			obj, _ := lhs.X.(*ast.Ident)
			if obj == nil {
				return true
			}
			owner := locals[obj.Name]
			if owner == "" {
				return true
			}
			rhsName := ""
			switch rhs := as.Rhs[0].(type) {
			case *ast.Ident:
				rhsName = rhs.Name
			case *ast.SelectorExpr:
				if id, ok := rhs.X.(*ast.Ident); ok {
					if t := strings.TrimPrefix(locals[id.Name], "*"); t != "" {
						rhsName = t + "." + rhs.Sel.Name
					}
				}
				if rhsName == "" {
					rhsName = selText(rhs)
				}
			}
			if rhsName == "" {
				return true
			}
			fields[dir+"|"+owner+"|"+lhs.Sel.Name+"=stored"] = rhsName
			valueBinds[obj.Name+"."+lhs.Sel.Name] = rhsName
			return true
		})
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			line := fset.Position(call.Pos()).Line
			text := selText(sel)
			parts := strings.Split(text, ".")
			if len(parts) < 2 {
				return true
			}
			cur := locals[parts[0]]
			if cur == "" {
				cur = pkgVars[parts[0]]
			}
			cur = strings.TrimPrefix(cur, "*")
			for i := 1; i < len(parts)-1 && cur != ""; i++ {
				cur = strings.TrimPrefix(fields[dir+"|"+cur+"|"+parts[i]], "*")
			}
			method := parts[len(parts)-1]
			if cur == "" {
				return true
			}
			if ifaces[dir+"|"+cur][method] {
				target := ix.lookupMethod(dir, cur, method)
				ix.addResolved(from, line, text, target, "interface", "interface-inferred", "receiver "+cur+" is an interface")
				for _, concrete := range concretes[method] {
					if concrete == cur {
						continue
					}
					impl := ix.lookupMethod(dir, concrete, method)
					ix.addResolved(from, line, text, impl, "interface", "interface-inferred", "interface "+cur+" candidate "+concrete)
				}
				return true
			}
			if stored := fields[dir+"|"+cur+"|"+method+"=stored"]; stored != "" {
				target := lookupBound(stored)
				ix.addResolved(from, line, text, target, "method-value", "statically-confirmed", "stored method value")
				return true
			}
			if bound := valueBinds[text]; bound != "" {
				target := lookupBound(bound)
				ix.addResolved(from, line, text, target, "method-value", "statically-confirmed", "stored method value invoked through "+text)
				return true
			}
			if ix.methodExists(dir, cur, method) {
				target := ix.lookupMethod(dir, cur, method)
				ix.addResolved(from, line, text, target, "method", "statically-confirmed", "receiver type "+cur)
			} else if strings.HasPrefix(fields[dir+"|"+cur+"|"+method], "func") {
				ix.addResolved(from, line, text, symbol{}, "stored-func", "unresolved", "func field has no stored value")
			}
			return true
		})
	}
}

func (ix *index) addResolved(from symbol, line int, text string, target symbol, kind, confidence, reason string) {
	e := rawEdge{
		fromID: from.id, fromPath: from.path, fromName: from.name, fromLine: from.line,
		callLine: line, callee: text, kind: kind, confidence: confidence, reason: reason,
	}
	if target.id != "" {
		e.toID, e.toPath, e.toName, e.toLine = target.id, target.path, target.name, target.line
	}
	ix.edges = append(ix.edges, e)
}

func selText(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		base := selText(t.X)
		if base == "" {
			return t.Sel.Name
		}
		return base + "." + t.Sel.Name
	default:
		return ""
	}
}

func typeString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.SelectorExpr:
		return selText(t)
	case *ast.FuncType:
		return "func"
	default:
		return ""
	}
}
