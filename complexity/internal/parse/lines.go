package parse

import "strings"

// CodeLines reports, for each 1-based line, whether the line contains code.
// Comment-only and blank lines are false. lang is go, typescript, javascript,
// rust, python, sql, or prisma.
func CodeLines(src, lang string) []bool {
	lines := strings.Split(src, "\n")
	out := make([]bool, len(lines)+1)
	inBlock := false
	for i, line := range lines {
		var code bool
		code, inBlock = lineHasCode(line, inBlock, lang)
		out[i+1] = code
	}
	return out
}

type lineScan struct {
	line, lang    string
	i             int
	code, inBlock bool
	done          bool
}

func lineHasCode(line string, inBlock bool, lang string) (bool, bool) {
	s := lineScan{line: line, lang: lang, inBlock: inBlock}
	for s.i < len(s.line) && !s.done {
		s.step()
	}
	return s.code, s.inBlock
}

func (s *lineScan) step() {
	if s.inBlock {
		s.finishBlock()
		return
	}
	if isLineComment(s.line[s.i:], s.lang) {
		s.done = true
		return
	}
	if strings.HasPrefix(s.line[s.i:], "/*") {
		s.inBlock = true
		s.i += 2
		return
	}
	if s.quoted() {
		return
	}
	s.mark()
	s.i++
}

func (s *lineScan) finishBlock() {
	end := strings.Index(s.line[s.i:], "*/")
	if end < 0 {
		s.i = len(s.line)
		return
	}
	s.i += end + 2
	s.inBlock = false
}

func (s *lineScan) quoted() bool {
	if s.line[s.i] != '"' && s.line[s.i] != '\'' && s.line[s.i] != '`' {
		return false
	}
	s.code = true
	s.i = skipString(s.line, s.i)
	return true
}

func (s *lineScan) mark() {
	if s.line[s.i] != ' ' && s.line[s.i] != '\t' && s.line[s.i] != '\r' {
		s.code = true
	}
}

func isLineComment(s, lang string) bool {
	if strings.HasPrefix(s, "//") && lang != "python" && lang != "sql" && lang != "prisma" {
		return true
	}
	if strings.HasPrefix(s, "#") && (lang == "python" || lang == "prisma") {
		return true
	}
	if strings.HasPrefix(s, "--") && (lang == "sql") {
		return true
	}
	return false
}

func skipString(line string, i int) int {
	quote := line[i]
	i++
	for i < len(line) {
		if line[i] == '\\' && quote != '`' {
			i += 2
			continue
		}
		if line[i] == quote {
			return i + 1
		}
		i++
	}
	return i
}

func countBetween(code []bool, start, end int) int {
	if start < 1 {
		start = 1
	}
	n := 0
	for line := start; line <= end && line < len(code); line++ {
		if code[line] {
			n++
		}
	}
	return n
}
