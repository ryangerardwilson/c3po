package cyclotree

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ryangerardwilson/c3po/ast"
	"github.com/ryangerardwilson/c3po/vibetree"
)

type scoreHit struct {
	file, name  string
	line, cyclo int
}

type scoreIndex struct {
	byLine map[string]scoreHit
	list   []scoreHit
}

func openTree(root, query string) (*screen, error) {
	calls, err := ast.OpenCalls(root)
	if err != nil {
		return nil, err
	}
	fns, err := vibetree.Functions(root)
	if err != nil {
		return nil, err
	}
	scores := indexScores(fns)
	if query == "" {
		return fromEntry(root, calls, scores)
	}
	hits := calls.Resolve(query)
	switch len(hits) {
	case 0:
		return fromScores(scores, query)
	case 1:
		return fromCall(calls, scores, hits[0]), nil
	default:
		return nil, ambiguous(query, hitsToLines(hits))
	}
}

func fromCall(calls callees, scores *scoreIndex, h ast.Fn) *screen {
	n := &node{id: h.ID, name: h.Name, path: h.Path, line: h.Line, linked: true}
	n.cyclo, n.scored = scores.lookup(h.Path, h.Line, h.Name)
	s := &screen{root: n, calls: calls, scores: scores}
	s.bindDepth(n)
	s.expand(n)
	return s
}

func fromScores(scores *scoreIndex, query string) (*screen, error) {
	hits := scores.resolve(query)
	switch len(hits) {
	case 0:
		return nil, fmt.Errorf("cyclotree: no symbol %q", query)
	case 1:
		h := hits[0]
		n := &node{
			name: h.name, path: h.file, line: h.line,
			cyclo: h.cyclo, scored: true, loaded: true, open: true,
		}
		return &screen{root: n, scores: scores}, nil
	default:
		return nil, ambiguous(query, scoreLines(hits))
	}
}

type candidate struct {
	path, name string
	line       int
}

func hitsToLines(hits []ast.Fn) []candidate {
	out := make([]candidate, len(hits))
	for i, h := range hits {
		out[i] = candidate{path: h.Path, name: h.Name, line: h.Line}
	}
	return out
}

func scoreLines(hits []scoreHit) []candidate {
	out := make([]candidate, len(hits))
	for i, h := range hits {
		out[i] = candidate{path: h.file, name: h.name, line: h.line}
	}
	return out
}

func ambiguous(query string, hits []candidate) error {
	var b strings.Builder
	fmt.Fprintf(&b, "cyclotree: ambiguous %s\n", query)
	for _, h := range hits {
		fmt.Fprintf(&b, "  %s:%d %s\n", h.path, h.line, h.name)
	}
	return fmt.Errorf("%s", strings.TrimRight(b.String(), "\n"))
}

func indexScores(fns []vibetree.Function) *scoreIndex {
	ix := &scoreIndex{byLine: map[string]scoreHit{}}
	for _, fn := range fns {
		h := scoreHit{file: fn.File, name: fn.Name, line: fn.Line, cyclo: fn.Cyclomatic}
		ix.list = append(ix.list, h)
		ix.byLine[fn.File+"#"+strconv.Itoa(fn.Line)] = h
	}
	return ix
}

func (ix *scoreIndex) lookup(path string, line int, name string) (int, bool) {
	if ix == nil {
		return 0, false
	}
	if h, ok := ix.byLine[path+"#"+strconv.Itoa(line)]; ok {
		return h.cyclo, true
	}
	var found scoreHit
	n := 0
	for _, h := range ix.list {
		if h.file == path && (h.name == name || shortName(h.name) == name) {
			found = h
			n++
		}
	}
	if n == 1 {
		return found.cyclo, true
	}
	return 0, false
}

func (ix *scoreIndex) resolve(query string) []scoreHit {
	if ix == nil {
		return nil
	}
	pathHint, name := splitQuery(query)
	var out []scoreHit
	for _, h := range ix.list {
		if !pathMatches(h.file, pathHint) || !nameMatches(h.name, name) {
			continue
		}
		out = append(out, h)
	}
	return out
}

func pathMatches(file, hint string) bool {
	if hint == "" {
		return true
	}
	return file == hint || strings.HasSuffix(file, "/"+hint) || strings.Contains(file, hint)
}

func nameMatches(have, want string) bool {
	return have == want || shortName(have) == want
}

func splitQuery(q string) (string, string) {
	q = strings.TrimSpace(q)
	i := strings.LastIndex(q, ":")
	if i < 0 {
		return "", q
	}
	return strings.TrimPrefix(filepath.ToSlash(q[:i]), "./"), q[i+1:]
}

func shortName(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}
