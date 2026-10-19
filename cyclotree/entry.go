package cyclotree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/ast"
)

func fromEntry(root string, calls *ast.Calls, scores *scoreIndex) (*screen, error) {
	hits := entryHits(root, calls)
	rust := rustEntries(scores)
	switch {
	case len(hits)+len(rust) == 0:
		return nil, fmt.Errorf("cyclotree: no entrypoint")
	case len(hits)+len(rust) > 1:
		return nil, ambiguous("entrypoint", joinCandidates(hits, rust))
	case len(hits) == 1:
		s := fromCall(calls, scores, hits[0])
		s.expandReachable()
		return s, nil
	default:
		return fromScores(scores, rust[0].file+":"+rust[0].name)
	}
}

func joinCandidates(hits []ast.Fn, rust []scoreHit) []candidate {
	out := hitsToLines(hits)
	return append(out, scoreLines(rust)...)
}

func entryHits(root string, calls *ast.Calls) []ast.Fn {
	var out []ast.Fn
	for _, h := range calls.Resolve("main") {
		if isEntry(root, h) {
			out = append(out, h)
		}
	}
	return out
}

func isEntry(root string, h ast.Fn) bool {
	if h.Name != "main" {
		return false
	}
	switch strings.ToLower(filepath.Ext(h.Path)) {
	case ".go":
		return h.Kind == "func" && packageMain(root, h.Path)
	case ".py":
		return h.Kind == "def"
	case ".js", ".jsx", ".ts", ".tsx":
		return h.Kind == "function"
	default:
		return false
	}
}

func packageMain(root, rel string) bool {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "//") {
			continue
		}
		return trim == "package main" || strings.HasPrefix(trim, "package main ") || strings.HasPrefix(trim, "package main\t")
	}
	return false
}

func rustEntries(scores *scoreIndex) []scoreHit {
	if scores == nil {
		return nil
	}
	var out []scoreHit
	for _, h := range scores.list {
		if h.name == "main" && strings.HasSuffix(h.file, ".rs") {
			out = append(out, h)
		}
	}
	return out
}
