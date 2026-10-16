// Package pyimport reads Python import statements without running Python.
package pyimport

import (
	"regexp"
	"strings"
)

// Name is one imported name. Local is the binding in the importing file.
// Line is 1-based and is 0 when the name came from From alone.
type Name struct {
	Orig, Local string
	Line        int
}

// Import is one import statement. From is false for `import a.b`, whose
// single name is the dotted module.
type Import struct {
	Module string
	From   bool
	Names  []Name
}

var reFrom = regexp.MustCompile(`^from\s+(\S+)\s+import\s+(.+)$`)

// From parses one `from M import a, b as c` line. Parentheses, a trailing
// backslash, and a trailing comment are dropped.
func From(trim string) (Import, bool) {
	m := reFrom.FindStringSubmatch(trim)
	if len(m) != 3 {
		return Import{}, false
	}
	return Import{Module: m[1], From: true, Names: names(m[2], 0)}, true
}

// Scan lists the imports in Python source. A parenthesized or backslashed
// from-import keeps the names on later lines, each with its own line.
// Comments, quoted lines, and lines inside a triple-quoted string are not imports.
func Scan(src string) []Import {
	var s scanner
	for i, line := range strings.Split(src, "\n") {
		s.line(i+1, line)
	}
	return s.out
}

type scanner struct {
	out   []Import
	quote string // triple-quote delimiter still open
	until string // ")" or `\` while a from-import continues
}

func (s *scanner) line(n int, line string) {
	open := s.quote
	s.quote = tripleAfter(line, open)
	if open != "" {
		return
	}
	trim := strings.TrimSpace(line)
	switch {
	case s.until != "":
		s.more(n, trim)
	case strings.HasPrefix(trim, "from "):
		s.from(n, trim)
	case strings.HasPrefix(trim, "import "):
		s.plain(n, trim)
	}
}

func (s *scanner) from(n int, trim string) {
	imp, ok := From(trim)
	if !ok {
		return
	}
	for i := range imp.Names {
		imp.Names[i].Line = n
	}
	s.out = append(s.out, imp)
	s.until = continues(trim, "")
}

func (s *scanner) more(n int, trim string) {
	last := &s.out[len(s.out)-1]
	last.Names = append(last.Names, names(trim, n)...)
	s.until = continues(trim, s.until)
}

func (s *scanner) plain(n int, trim string) {
	for _, part := range strings.Split(code(strings.TrimPrefix(trim, "import ")), ",") {
		mod, local := alias(strings.TrimSpace(part))
		if mod == "" {
			continue
		}
		if local == mod {
			local = strings.SplitN(mod, ".", 2)[0]
		}
		s.out = append(s.out, Import{Module: mod, Names: []Name{{Orig: mod, Local: local, Line: n}}})
	}
}

// continues reports what still closes the statement after this line.
func continues(trim, until string) string {
	body := strings.TrimSpace(code(trim))
	if until == "" && strings.Contains(body, "(") {
		until = ")"
	}
	if until == ")" {
		if strings.Contains(body, ")") {
			return ""
		}
		return ")"
	}
	if strings.HasSuffix(body, `\`) {
		return `\`
	}
	return ""
}

func names(text string, line int) []Name {
	text = strings.NewReplacer("(", "", ")", "", `\`, "").Replace(code(text))
	var out []Name
	for _, part := range strings.Split(text, ",") {
		orig, local := alias(strings.TrimSpace(part))
		if orig != "" {
			out = append(out, Name{Orig: orig, Local: local, Line: line})
		}
	}
	return out
}

func alias(part string) (string, string) {
	if i := strings.Index(part, " as "); i >= 0 {
		return strings.TrimSpace(part[:i]), strings.TrimSpace(part[i+4:])
	}
	return part, part
}

// code drops a trailing comment. Import names hold no strings.
func code(text string) string {
	if i := strings.IndexByte(text, '#'); i >= 0 {
		return text[:i]
	}
	return text
}

// tripleAfter returns the triple-quote delimiter open at the end of line,
// given the one open at its start.
func tripleAfter(line, open string) string {
	i := 0
	for i < len(line) {
		if open != "" {
			j := strings.Index(line[i:], open)
			if j < 0 {
				return open
			}
			i += j + 3
			open = ""
			continue
		}
		rest := line[i:]
		switch {
		case rest[0] == '#':
			return ""
		case strings.HasPrefix(rest, `"""`) || strings.HasPrefix(rest, `'''`):
			open = rest[:3]
			i += 3
		case rest[0] == '"' || rest[0] == '\'':
			i = skipString(line, i)
		default:
			i++
		}
	}
	return open
}

func skipString(line string, i int) int {
	q := line[i]
	for i++; i < len(line); i++ {
		if line[i] == '\\' {
			i++
			continue
		}
		if line[i] == q {
			return i + 1
		}
	}
	return i
}
