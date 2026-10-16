package parse

import (
	"fmt"
	"hash/fnv"
	"strconv"

	"github.com/ryangerardwilson/c3po/vibetree/internal/metric"
)

// File scores the functions in src. lang is go, python, rust, typescript,
// or javascript. Unparsed files return a diagnostic and no functions.
func File(path, lang string, src []byte) ([]metric.Function, []string) {
	switch lang {
	case "go":
		return ParseGo(path, src)
	case "python":
		return ParsePython(path, src)
	case "rust":
		return parseBrace(path, "rust", "rust", src)
	case "typescript":
		return parseBrace(path, "ts", "typescript", src)
	case "javascript":
		return parseBrace(path, "ts", "javascript", src)
	default:
		return nil, nil
	}
}

func parseBrace(path, lexLang, label string, src []byte) ([]metric.Function, []string) {
	toks := lex(string(src), lexLang)
	spans := findFunctions(toks, lexLang)
	inners := indexInners(spans)
	code := CodeLines(string(src), label)
	names := anonNames(spans)
	out := make([]metric.Function, 0, len(spans))
	for k, s := range spans {
		sc := scoreSpan(toks, s, inners, lexLang)
		fn := metric.Function{
			File: path, Line: s.line, EndLine: s.endLine, Name: names[k],
			Language: label, Cognitive: sc.cog, Nesting: sc.nest,
			Lines:  countBetween(code, s.line, s.endLine),
			Params: s.params, Cyclomatic: sc.cyclo, Events: sc.events,
		}
		if metric.Anonymous(fn.Name) {
			fn.Sig = tokenSig(toks[s.start:s.end])
		}
		out = append(out, fn)
	}
	return out, nil
}

// anonNames suffixes a second anonymous function on one line with #2.
func anonNames(spans []span) []string {
	seen := map[string]int{}
	out := make([]string, len(spans))
	for k, s := range spans {
		out[k] = s.name
		if !metric.Anonymous(s.name) {
			continue
		}
		seen[s.name]++
		if n := seen[s.name]; n > 1 {
			out[k] = fmt.Sprintf("%s#%d", s.name, n)
		}
	}
	return out
}

// tokenSig hashes a function's tokens without line breaks, so moving the
// function does not change it.
func tokenSig(toks []Token) string {
	h := fnv.New64a()
	for _, t := range toks {
		if t.Lit != "nl" {
			_, _ = h.Write([]byte(t.Lit + "\x00"))
		}
	}
	return strconv.FormatUint(h.Sum64(), 16)
}
