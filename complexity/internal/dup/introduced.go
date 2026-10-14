package dup

import "strings"

// Hit is a run of changed lines that repeats code that already existed.
type Hit struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	OfFile string `json:"of_file"`
	OfLine int    `json:"of_line"`
	Lines  int    `json:"lines"`
}

// Fresh reports whether a line of a changed file is new.
type Fresh func(line int) bool

// Body is the first and last line of one function.
type Body [2]int

type place struct {
	file string
	line int
	idx  int
}

// Introduced finds windows of min literal lines inside function bodies of
// changed files that hold a new line and repeat a window of lines that
// already existed in pool. Pool includes the changed files. Lines shorter
// than four characters and comments are skipped.
func Introduced(changed map[string]Fresh, pool []File, bodies map[string][]Body, min int) []Hit {
	if min < 1 {
		min = 6
	}
	lines := map[string][]normLine{}
	for _, f := range pool {
		lines[f.Path] = literal(f, bodies[f.Path])
	}
	old := existing(changed, pool, lines, min)
	var out []Hit
	for _, f := range pool {
		if fresh, ok := changed[f.Path]; ok {
			out = append(out, copiesIn(lines[f.Path], fresh, old, min)...)
		}
	}
	return out
}

func existing(changed map[string]Fresh, pool []File, lines map[string][]normLine, min int) map[uint64]place {
	out := map[uint64]place{}
	for _, f := range pool {
		ls := lines[f.Path]
		fresh := changed[f.Path]
		for i := 0; i+min <= len(ls); i++ {
			if anyFresh(ls[i:i+min], fresh) {
				continue
			}
			h := window(ls[i : i+min])
			if _, ok := out[h]; !ok {
				out[h] = place{f.Path, ls[i].line, i}
			}
		}
	}
	return out
}

func copiesIn(ls []normLine, fresh Fresh, old map[uint64]place, min int) []Hit {
	var out []Hit
	last := -min
	for i := 0; i+min <= len(ls); i++ {
		if !anyFresh(ls[i:i+min], fresh) {
			continue
		}
		p, ok := old[window(ls[i:i+min])]
		if !ok || (p.file == ls[i].file && abs(p.idx-i) < min) {
			continue
		}
		if len(out) > 0 && i-last < min {
			out[len(out)-1].Lines = ls[i+min-1].line - out[len(out)-1].Line + 1
		} else {
			out = append(out, Hit{File: ls[i].file, Line: ls[i].line, OfFile: p.file, OfLine: p.line, Lines: ls[i+min-1].line - ls[i].line + 1})
		}
		last = i
	}
	return out
}

func anyFresh(ls []normLine, fresh Fresh) bool {
	if fresh == nil {
		return false
	}
	for _, l := range ls {
		if fresh(l.line) {
			return true
		}
	}
	return false
}

// literal keeps trimmed code lines inside a function body. A nil body list
// keeps every line.
func literal(f File, bodies []Body) []normLine {
	var out []normLine
	for i, line := range strings.Split(f.Src, "\n") {
		text := strings.TrimSpace(line)
		if len(text) < 4 || skipNorm(text) || !inBody(i+1, bodies) {
			continue
		}
		out = append(out, normLine{file: f.Path, line: i + 1, text: text})
	}
	return out
}

func inBody(line int, bodies []Body) bool {
	if bodies == nil {
		return true
	}
	for _, b := range bodies {
		if line > b[0] && line < b[1] {
			return true
		}
	}
	return false
}
