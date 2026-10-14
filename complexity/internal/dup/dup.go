package dup

import (
	"fmt"
	"hash/fnv"
	"strings"
)

// Occurrence is one copy of a clone.
type Occurrence struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// Cluster is one Type-2 clone of at least Min lines.
type Cluster struct {
	ID          string       `json:"id"`
	Lines       int          `json:"lines"`
	Similarity  float64      `json:"similarity"`
	Occurrences []Occurrence `json:"occurrences"`
}

// File is source text to scan for clones.
type File struct {
	Path string
	Src  string
	Lang string
}

type normLine struct {
	file string
	line int
	text string
}

type win struct {
	file string
	line int
}

// Find reports normalized clones of at least min lines.
func Find(files []File, min int) []Cluster {
	if min < 1 {
		min = 6
	}
	lines := allLines(files)
	if len(lines) < min*2 {
		return nil
	}
	return clusters(windows(lines, min), min)
}

func allLines(files []File) []normLine {
	var lines []normLine
	for _, f := range files {
		lines = append(lines, normalize(f)...)
	}
	return lines
}

func windows(lines []normLine, min int) map[uint64][]win {
	hits := map[uint64][]win{}
	for i := 0; i+min <= len(lines); i++ {
		if lines[i].file != lines[i+min-1].file {
			continue
		}
		h := window(lines[i : i+min])
		hits[h] = append(hits[h], win{lines[i].file, lines[i].line})
	}
	return hits
}

func clusters(hits map[uint64][]win, min int) []Cluster {
	var out []Cluster
	seen := map[string]bool{}
	for _, group := range hits {
		if len(out) >= 40 {
			break
		}
		c, key, ok := oneCluster(group, min, len(out)+1)
		if !ok || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}
	return out
}

func oneCluster(group []win, min, n int) (Cluster, string, bool) {
	if len(group) < 2 || len(group) > 12 {
		return Cluster{}, "", false
	}
	a, b := group[0], group[1]
	if a.file == b.file && abs(a.line-b.line) < min {
		return Cluster{}, "", false
	}
	key := pairKey(a.file, a.line, b.file, b.line)
	c := Cluster{
		ID: fmt.Sprintf("dup-%03d", n), Lines: min, Similarity: 1,
		Occurrences: []Occurrence{{a.file, a.line}, {b.file, b.line}},
	}
	return c, key, true
}

func normalize(f File) []normLine {
	var out []normLine
	for i, line := range strings.Split(f.Src, "\n") {
		text := norm(line)
		if text == "" {
			continue
		}
		out = append(out, normLine{file: f.Path, line: i + 1, text: text})
	}
	return out
}

func norm(line string) string {
	line = strings.TrimSpace(line)
	if skipNorm(line) {
		return ""
	}
	var b strings.Builder
	for i := 0; i < len(line); {
		i = normByte(&b, line, i)
	}
	return dropBrace(b.String())
}

func skipNorm(line string) bool {
	if line == "" || len(line) > 240 {
		return true
	}
	return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "--")
}

func normByte(b *strings.Builder, line string, i int) int {
	c := line[i]
	if isIdent(c) {
		return writeRun(b, line, i, 'i', identRun)
	}
	if c >= '0' && c <= '9' {
		return writeRun(b, line, i, '0', digitRun)
	}
	if c == '"' || c == '\'' || c == '`' {
		return writeString(b, line, i)
	}
	if c == ' ' || c == '\t' {
		return i + 1
	}
	b.WriteByte(c)
	return i + 1
}

func writeRun(b *strings.Builder, line string, i int, mark byte, ok func(byte) bool) int {
	b.WriteByte(mark)
	i++
	for i < len(line) && ok(line[i]) {
		i++
	}
	return i
}

func identRun(c byte) bool {
	return isIdent(c) || (c >= '0' && c <= '9')
}

func digitRun(c byte) bool {
	return c >= '0' && c <= '9'
}

func writeString(b *strings.Builder, line string, i int) int {
	q := line[i]
	b.WriteByte('s')
	i++
	for i < len(line) && line[i] != q {
		i++
	}
	if i < len(line) {
		i++
	}
	return i
}

func dropBrace(text string) string {
	if text == "{" || text == "}" || text == "()" {
		return ""
	}
	return text
}

func isIdent(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func window(lines []normLine) uint64 {
	h := fnv.New64a()
	for _, ln := range lines {
		_, _ = h.Write([]byte(ln.text))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}

func pairKey(f1 string, l1 int, f2 string, l2 int) string {
	if f2 < f1 || (f2 == f1 && l2 < l1) {
		f1, l1, f2, l2 = f2, l2, f1, l1
	}
	return fmt.Sprintf("%s:%d:%s:%d", f1, l1, f2, l2)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
