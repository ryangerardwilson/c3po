package models

import "strings"

func parseSQL(file, src string) []draft {
	clean, lines := stripSQL(src)
	low := strings.ToLower(clean)
	var out []draft
	from := 0
	for {
		i := strings.Index(low[from:], "create table")
		if i < 0 {
			break
		}
		i += from
		d, next := oneTable(file, clean, lines, i)
		if d.name != "" {
			out = append(out, d)
		}
		if next <= i {
			next = i + len("create table")
		}
		from = next
	}
	applyAlters(clean, out)
	return out
}

func oneTable(file, src string, lineAt []int, i int) (draft, int) {
	rest := src[i+len("create table"):]
	j := 0
	j = skipWords(rest, j, "if", "not", "exists")
	name, j := readIdent(rest, j)
	if name == "" {
		return draft{}, i + len("create table")
	}
	k := strings.Index(rest[j:], "(")
	if k < 0 {
		return draft{}, i + len("create table")
	}
	j += k
	body, end := parenBody(rest, j)
	d := draft{name: name, file: file, line: lineAt[min(i, len(lineAt)-1)]}
	for _, part := range splitComma(body) {
		addSQLPart(&d, strings.TrimSpace(part), d.line)
	}
	absEnd := i + len("create table") + end
	if absEnd > len(src) {
		absEnd = len(src)
	}
	return d, absEnd
}

func addSQLPart(d *draft, part string, line int) {
	if part == "" {
		return
	}
	low := strings.ToLower(part)
	switch {
	case strings.HasPrefix(low, "foreign key"):
		addSQLFK(d, low)
	case strings.HasPrefix(low, "primary key"):
		markPK(d, low)
	case strings.HasPrefix(low, "constraint"):
		if strings.Contains(low, "foreign key") {
			addSQLFK(d, low)
		}
	default:
		addSQLColumn(d, part, low, line)
	}
}

func addSQLColumn(d *draft, part, low string, line int) {
	name, _ := readIdent(part, 0)
	if name == "" {
		return
	}
	col := Column{Name: name, Line: line, Role: sqlRole(name, low)}
	if strings.Contains(low, "references") {
		col.Role = "fk"
		if target := refTarget(low); target != "" {
			d.rels = append(d.rels, Relationship{Name: name, Target: target, Type: "ForeignKey"})
		}
	}
	if strings.Contains(low, "primary key") {
		col.Role = "pk"
	}
	d.columns = append(d.columns, col)
}

func addSQLFK(d *draft, low string) {
	col := identInParens(low)
	target := refTarget(low)
	if col != "" {
		for i := range d.columns {
			if strings.EqualFold(d.columns[i].Name, col) {
				d.columns[i].Role = "fk"
			}
		}
	}
	if target != "" {
		d.rels = append(d.rels, Relationship{Name: col, Target: target, Type: "ForeignKey"})
	}
}

func markPK(d *draft, low string) {
	col := identInParens(low)
	for i := range d.columns {
		if col == "" || strings.EqualFold(d.columns[i].Name, col) {
			d.columns[i].Role = "pk"
			if col != "" {
				return
			}
		}
	}
}

func sqlRole(name, low string) string {
	lname := strings.ToLower(name)
	switch {
	case strings.Contains(low, "primary key") || lname == "id":
		return "pk"
	case strings.HasSuffix(lname, "_id"):
		return "fk"
	case isTimestamp(lname):
		return "timestamp"
	case lname == "data":
		return "data"
	default:
		return "feature"
	}
}

func isTimestamp(name string) bool {
	switch name {
	case "created_at", "updated_at", "deleted_at", "createdat", "updatedat", "deletedat":
		return true
	default:
		return strings.HasSuffix(name, "_at")
	}
}

func refTarget(low string) string {
	i := strings.Index(low, "references")
	if i < 0 {
		return ""
	}
	name, _ := readIdent(low[i+len("references"):], 0)
	return name
}

func identInParens(s string) string {
	i := strings.Index(s, "(")
	if i < 0 {
		return ""
	}
	body, _ := parenBody(s, i)
	name, _ := readIdent(body, 0)
	return name
}

func applyAlters(src string, drafts []draft) {
	low := strings.ToLower(src)
	from := 0
	for {
		i := strings.Index(low[from:], "alter table")
		if i < 0 {
			return
		}
		i += from
		rest := src[i+len("alter table"):]
		name, j := readIdent(rest, 0)
		clause := strings.ToLower(rest[j:])
		if name != "" && strings.Contains(clause, "foreign key") && strings.Contains(clause, "references") {
			for di := range drafts {
				if strings.EqualFold(drafts[di].name, name) {
					addSQLFK(&drafts[di], clause)
				}
			}
		}
		from = i + len("alter table")
	}
}

func skipWords(s string, i int, words ...string) int {
	for _, w := range words {
		j := skipSpace(s, i)
		if hasWord(s[j:], w) {
			i = j + len(w)
			continue
		}
		return i
	}
	return i
}

func hasWord(s, w string) bool {
	if len(s) < len(w) || !strings.EqualFold(s[:len(w)], w) {
		return false
	}
	if len(s) == len(w) {
		return true
	}
	return !isIdentByte(s[len(w)])
}

func readIdent(s string, i int) (string, int) {
	i = skipSpace(s, i)
	if i >= len(s) {
		return "", i
	}
	if s[i] == '"' || s[i] == '`' || s[i] == '\'' || s[i] == '[' {
		return readQuoted(s, i)
	}
	j := i
	for j < len(s) && (isIdentByte(s[j]) || s[j] == '.') {
		j++
	}
	name := s[i:j]
	if k := strings.LastIndex(name, "."); k >= 0 {
		name = name[k+1:]
	}
	return name, j
}

func readQuoted(s string, i int) (string, int) {
	q := s[i]
	end := byte('"')
	if q == '`' {
		end = '`'
	}
	if q == '\'' {
		end = '\''
	}
	if q == '[' {
		end = ']'
	}
	j := i + 1
	for j < len(s) && s[j] != end {
		j++
	}
	name := s[i+1 : min(j, len(s))]
	if j < len(s) {
		j++
	}
	return name, j
}

func parenBody(s string, open int) (string, int) {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[open+1 : i], i + 1
			}
		}
	}
	return s[open+1:], len(s)
}

func splitComma(body string) []string {
	var out []string
	depth := 0
	start := 0
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				out = append(out, body[start:i])
				start = i + 1
			}
		}
	}
	return append(out, body[start:])
}

func skipSpace(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	return i
}

func isIdentByte(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func stripSQL(src string) (string, []int) {
	var b strings.Builder
	lineAt := make([]int, 0, len(src))
	line := 1
	for i := 0; i < len(src); {
		if strings.HasPrefix(src[i:], "--") {
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if strings.HasPrefix(src[i:], "/*") {
			i = skipBlock(src, i)
			continue
		}
		if src[i] == '\n' {
			line++
		}
		b.WriteByte(src[i])
		lineAt = append(lineAt, line)
		i++
	}
	return b.String(), lineAt
}

func skipBlock(src string, i int) int {
	end := strings.Index(src[i+2:], "*/")
	if end < 0 {
		return len(src)
	}
	return i + 2 + end + 2
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
