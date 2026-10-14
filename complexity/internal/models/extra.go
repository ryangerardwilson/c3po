package models

import "strings"

func parseTS(file, src string) []draft {
	if strings.Contains(src, "@Entity") {
		return parseEntities(file, src)
	}
	return parseDrizzle(file, src)
}

func parseEntities(file, src string) []draft {
	lines := strings.Split(src, "\n")
	var out []draft
	for i := 0; i < len(lines); i++ {
		if !strings.Contains(lines[i], "@Entity") {
			continue
		}
		name, at := nextClass(lines, i)
		if name == "" {
			continue
		}
		body, next := braceBlock(lines, at)
		out = append(out, entityDraft(file, name, at+1, body))
		i = next
	}
	return out
}

func nextClass(lines []string, from int) (string, int) {
	for i := from; i < len(lines) && i < from+6; i++ {
		trim := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trim, "class ") {
			name, _, ok := classAt(strings.TrimPrefix(trim, "export "))
			if !ok {
				name = strings.Fields(strings.TrimPrefix(trim, "export "))[1]
			}
			return name, i
		}
	}
	return "", from
}

func entityDraft(file, name string, line int, body []string) draft {
	d := draft{name: name, file: file, line: line}
	var decor []string
	for i, lineText := range body {
		trim := strings.TrimSpace(stripLineComment(lineText))
		if strings.HasPrefix(trim, "@") {
			decor = append(decor, trim)
			continue
		}
		addEntityField(&d, trim, strings.Join(decor, " "), line+i)
		decor = nil
	}
	return d
}

func addEntityField(d *draft, line, decor string, n int) {
	name, typ, ok := tsField(line)
	if !ok {
		return
	}
	switch {
	case strings.Contains(decor, "@ManyToMany"):
		d.rels = append(d.rels, Relationship{Name: name, Target: typ, Type: "ManyToMany"})
	case strings.Contains(decor, "@OneToMany"):
		d.rels = append(d.rels, Relationship{Name: name, Target: typ, Type: "reverse_FK"})
	case strings.Contains(decor, "@ManyToOne") || strings.Contains(decor, "@OneToOne"):
		d.rels = append(d.rels, Relationship{Name: name, Target: typ, Type: "ForeignKey"})
	default:
		d.columns = append(d.columns, Column{Name: name, Role: entityRole(name, decor), Line: n})
	}
}

func entityRole(name, decor string) string {
	low := toSnake(name)
	switch {
	case strings.Contains(decor, "@Primary") || low == "id":
		return "pk"
	case strings.HasSuffix(low, "_id"):
		return "fk"
	case isTimestamp(low):
		return "timestamp"
	case low == "data":
		return "data"
	default:
		return "feature"
	}
}

func tsField(line string) (string, string, bool) {
	line = strings.TrimSuffix(strings.TrimSpace(line), ";")
	if line == "" || strings.HasPrefix(line, "class ") || line == "}" {
		return "", "", false
	}
	if i := strings.Index(line, ":"); i > 0 {
		name := strings.TrimSpace(line[:i])
		typ := arrowType(line[i+1:])
		if plainIdent(name) {
			return name, typ, true
		}
	}
	f := strings.Fields(line)
	if len(f) == 1 && plainIdent(f[0]) {
		return f[0], "", true
	}
	return "", "", false
}

func arrowType(s string) string {
	if name := identAfter(s, "=>"); name != "" {
		return name
	}
	f := strings.Fields(s)
	if len(f) == 0 {
		return ""
	}
	return strings.Trim(f[0], "[]?;")
}

func identAfter(s, mark string) string {
	i := strings.Index(s, mark)
	if i < 0 {
		return ""
	}
	name, _ := readIdent(s[i+len(mark):], 0)
	return name
}

func parseDrizzle(file, src string) []draft {
	lines := strings.Split(src, "\n")
	var out []draft
	for i, line := range lines {
		name, ok := drizzleTable(line)
		if !ok {
			continue
		}
		body, next := braceBlock(lines, i)
		out = append(out, drizzleDraft(file, name, i+1, body))
		i = next
	}
	return out
}

func drizzleTable(line string) (string, bool) {
	for _, fn := range []string{"pgTable", "mysqlTable", "sqliteTable"} {
		if strings.Contains(line, fn+"(") {
			if name := quotedArg(line); name != "" {
				return name, true
			}
		}
	}
	return "", false
}

func drizzleDraft(file, name string, line int, body []string) draft {
	d := draft{name: name, table: name, file: file, line: line}
	for i, lineText := range body {
		addDrizzle(&d, lineText, line+i)
	}
	return d
}

func addDrizzle(d *draft, line string, n int) {
	key, rest, ok := assign(line)
	if !ok {
		return
	}
	col := quotedArg(rest)
	if col == "" {
		col = toSnake(key)
	}
	role := "feature"
	low := strings.ToLower(rest)
	switch {
	case strings.Contains(low, "primarykey"):
		role = "pk"
	case strings.Contains(low, "references"):
		role = "fk"
		if target := identAfter(rest, "=>"); target != "" {
			d.rels = append(d.rels, Relationship{Name: col, Target: target, Type: "ForeignKey"})
		}
	case isTimestamp(col):
		role = "timestamp"
	case col == "data":
		role = "data"
	}
	d.columns = append(d.columns, Column{Name: col, Role: role, Line: n})
}

func parseRust(file, src string) []draft {
	var out []draft
	out = append(out, rustTables(file, src)...)
	out = append(out, rustEntities(file, src)...)
	return out
}

func rustTables(file, src string) []draft {
	if !strings.Contains(src, "table!") {
		return nil
	}
	lines := strings.Split(src, "\n")
	var out []draft
	for i, line := range lines {
		if !strings.Contains(line, "table!") {
			continue
		}
		body, next := braceBlock(lines, i)
		if d, ok := rustTable(file, i+1, body); ok {
			out = append(out, d)
		}
		i = next
	}
	applyJoinable(src, out)
	return out
}

func rustTable(file string, line int, body []string) (draft, bool) {
	if len(body) == 0 {
		return draft{}, false
	}
	name, _ := readIdent(strings.TrimSpace(body[0]), 0)
	if name == "" {
		return draft{}, false
	}
	d := draft{name: name, file: file, line: line}
	for i, lineText := range body {
		left := strings.TrimSpace(lineText)
		if strings.Contains(left, "->") {
			col, _ := readIdent(left, 0)
			if col != "" {
				d.columns = append(d.columns, Column{Name: col, Role: sqlRole(col, left), Line: line + i})
			}
		}
	}
	return d, true
}

func applyJoinable(src string, drafts []draft) {
	low := strings.ToLower(src)
	from := 0
	for {
		i := strings.Index(low[from:], "joinable!")
		if i < 0 {
			return
		}
		i += from
		rest := low[i:]
		child, j := readIdent(rest[len("joinable!"):], 0)
		if j < len(rest) {
			rest = rest[len("joinable!")+j:]
		}
		if k := strings.Index(rest, "->"); k >= 0 {
			parent, _ := readIdent(rest[k+2:], 0)
			for di := range drafts {
				if strings.EqualFold(drafts[di].name, child) && parent != "" {
					drafts[di].rels = append(drafts[di].rels, Relationship{Name: parent, Target: parent, Type: "ForeignKey"})
				}
			}
		}
		from = i + len("joinable!")
	}
}

func rustEntities(file, src string) []draft {
	if !strings.Contains(src, "sea_orm") && !strings.Contains(src, "belongs_to") {
		return nil
	}
	lines := strings.Split(src, "\n")
	var out []draft
	var pending string
	for i, line := range lines {
		if strings.Contains(line, "table_name") {
			pending = quotedArg(line)
		}
		name, ok := structAt(line)
		if !ok {
			continue
		}
		body, next := goStruct(lines, i)
		d := draft{name: name, table: pending, file: file, line: i + 1}
		if strings.Contains(body, "belongs_to") || pending != "" {
			addBelongs(&d, body)
			out = append(out, d)
		}
		pending = ""
		i = next
	}
	return out
}

func addBelongs(d *draft, body string) {
	low := strings.ToLower(body)
	i := strings.Index(low, "belongs_to")
	if i < 0 {
		return
	}
	target := quotedArg(body[i:])
	if k := strings.LastIndex(target, "::"); k >= 0 {
		target = target[k+2:]
	}
	if target != "" && target != "Entity" {
		d.rels = append(d.rels, Relationship{Name: "belongs_to", Target: target, Type: "ForeignKey"})
	}
}
