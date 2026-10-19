package trace

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

func (ix *index) resolveGoImports() {
	dirs := ix.goDirs()
	imps := map[string]map[string]string{}
	for _, rel := range ix.goFiles() {
		imps[rel] = goFileImports(filepath.Join(ix.root, filepath.FromSlash(rel)))
	}
	for i := range ix.edges {
		ix.linkGoImport(&ix.edges[i], imps, dirs)
	}
}

func (ix *index) goFiles() []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range ix.syms {
		if !strings.HasSuffix(s.path, ".go") || seen[s.path] {
			continue
		}
		seen[s.path] = true
		out = append(out, s.path)
	}
	return out
}

func (ix *index) goDirs() []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range ix.syms {
		if !strings.HasSuffix(s.path, ".go") || seen[s.dir] {
			continue
		}
		seen[s.dir] = true
		out = append(out, s.dir)
	}
	return out
}

func (ix *index) linkGoImport(e *rawEdge, imps map[string]map[string]string, dirs []string) {
	if e.toID != "" || e.kind != "selector" || e.confidence != "unresolved" {
		return
	}
	pkg, name, ok := pkgCall(e.callee)
	if !ok {
		return
	}
	dir := dirForImport(dirs, imps[e.fromPath][pkg])
	if dir == "" {
		return
	}
	target, n := ix.funcInDir(dir, name)
	if n != 1 {
		return
	}
	e.toID, e.toPath, e.toName, e.toLine = target.id, target.path, target.name, target.line
	e.confidence = "statically-confirmed"
}

func (ix *index) funcInDir(dir, name string) (symbol, int) {
	var found symbol
	n := 0
	for _, s := range ix.syms {
		if s.dir == dir && s.kind == "func" && s.name == name {
			found = s
			n++
		}
	}
	return found, n
}

func pkgCall(callee string) (string, string, bool) {
	pkg, name, ok := strings.Cut(callee, ".")
	if !ok || pkg == "" || name == "" || strings.Contains(name, ".") {
		return "", "", false
	}
	return pkg, name, true
}

func dirForImport(dirs []string, path string) string {
	if path == "" || !strings.Contains(path, "/") {
		return ""
	}
	best, tie := longestImportDir(dirs, path)
	if tie {
		return ""
	}
	return best
}

func longestImportDir(dirs []string, path string) (string, bool) {
	best := ""
	tie := false
	for _, dir := range dirs {
		if !importDirMatch(dir, path) {
			continue
		}
		if len(dir) > len(best) {
			best = dir
			tie = false
			continue
		}
		if len(dir) == len(best) {
			tie = true
		}
	}
	return best, tie
}

func importDirMatch(dir, path string) bool {
	if dir == "." || dir == "" {
		return false
	}
	return path == dir || strings.HasSuffix(path, "/"+dir)
}

func goFileImports(abs string) map[string]string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, abs, nil, parser.ImportsOnly)
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for _, imp := range f.Imports {
		local, path, ok := importLocal(imp)
		if ok {
			out[local] = path
		}
	}
	return out
}

func importLocal(imp *ast.ImportSpec) (string, string, bool) {
	path, err := strconv.Unquote(imp.Path.Value)
	if err != nil || path == "" {
		return "", "", false
	}
	local := path
	if imp.Name != nil {
		local = imp.Name.Name
	} else if slash := strings.LastIndex(path, "/"); slash >= 0 {
		local = path[slash+1:]
	}
	if local == "" || local == "_" || local == "." {
		return "", "", false
	}
	return local, path, true
}
