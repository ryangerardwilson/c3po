package models

import "strings"

func parsePython(file, src string) []draft {
	lines := strings.Split(src, "\n")
	var out []draft
	for i := 0; i < len(lines); i++ {
		name, indent, ok := classAt(lines[i])
		if !ok {
			continue
		}
		body, next := indented(lines, i+1, indent)
		if isPyModel(body) {
			out = append(out, pyDraft(file, name, i+1, body))
		}
		if next > i {
			i = next - 1
		}
	}
	return out
}

func classAt(line string) (string, int, bool) {
	trim := strings.TrimSpace(stripHash(line))
	if !strings.HasPrefix(trim, "class ") {
		return "", 0, false
	}
	name := strings.TrimPrefix(trim, "class ")
	if i := strings.IndexAny(name, "(:"); i >= 0 {
		name = name[:i]
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", 0, false
	}
	return name, indentOf(line), true
}

func indented(lines []string, start, header int) (string, int) {
	var b strings.Builder
	i := start
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "" {
			b.WriteByte('\n')
			i++
			continue
		}
		if indentOf(lines[i]) <= header {
			break
		}
		b.WriteString(lines[i])
		b.WriteByte('\n')
		i++
	}
	return b.String(), i
}

func indentOf(line string) int {
	n := 0
	for _, c := range line {
		if c == ' ' {
			n++
			continue
		}
		if c == '\t' {
			n += 8
			continue
		}
		break
	}
	return n
}

func isPyModel(body string) bool {
	keys := []string{"__tablename__", "table=True", "table = True", "ForeignKey", "foreign_key", "relationship(", "Column(", "mapped_column("}
	for _, k := range keys {
		if strings.Contains(body, k) {
			return true
		}
	}
	return false
}

func pyDraft(file, name string, line int, body string) draft {
	d := draft{name: name, file: file, line: line}
	var rels []Relationship
	var fkEdges []Relationship
	for i, lineText := range strings.Split(body, "\n") {
		if tab := tableName(lineText); tab != "" {
			d.table = tab
			continue
		}
		col, rel, fk, ok := pyLine(lineText, line+1+i)
		if !ok {
			continue
		}
		if col.Name != "" {
			d.columns = append(d.columns, col)
		}
		if rel.Target != "" {
			rels = append(rels, rel)
		}
		if fk.Target != "" {
			fkEdges = append(fkEdges, fk)
		}
	}
	if len(fkEdges) > 0 {
		for i := range rels {
			rels[i].Type = "ForeignKey"
		}
	}
	if len(rels) == 0 {
		rels = fkEdges
	}
	d.rels = rels
	return d
}

func tableName(line string) string {
	if !strings.Contains(line, "__tablename__") {
		return ""
	}
	return quotedArg(line)
}

func pyLine(line string, n int) (Column, Relationship, Relationship, bool) {
	text := stripHash(line)
	if strings.TrimSpace(text) == "" || strings.Contains(text, "__tablename__") {
		return Column{}, Relationship{}, Relationship{}, false
	}
	name, rhs, ok := assign(text)
	if !ok {
		return Column{}, Relationship{}, Relationship{}, false
	}
	if strings.Contains(rhs, "relationship(") {
		return Column{}, Relationship{Name: name, Target: quotedArg(rhs), Type: "reverse_FK"}, Relationship{}, true
	}
	col := Column{Name: name, Line: n, Role: pyRole(name, rhs)}
	var fk Relationship
	if strings.Contains(rhs, "ForeignKey") || strings.Contains(rhs, "foreign_key") {
		col.Role = "fk"
		if target := fkTarget(quotedArg(rhs)); target != "" {
			fk = Relationship{Name: name, Target: target, Type: "ForeignKey"}
		}
	}
	if strings.Contains(rhs, "primary_key=True") || strings.Contains(rhs, "primary_key = True") {
		col.Role = "pk"
	}
	return col, Relationship{}, fk, true
}

func assign(line string) (string, string, bool) {
	i := strings.Index(line, "=")
	if i < 0 {
		return "", "", false
	}
	lhs := strings.TrimSpace(line[:i])
	if c := strings.Index(lhs, ":"); c >= 0 {
		lhs = strings.TrimSpace(lhs[:c])
	}
	if !plainIdent(lhs) {
		return "", "", false
	}
	return lhs, line[i+1:], true
}

func pyRole(name, rhs string) string {
	low := strings.ToLower(name)
	switch {
	case low == "id" || strings.Contains(rhs, "primary_key"):
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

func fkTarget(arg string) string {
	arg = strings.Trim(arg, `"'`)
	if i := strings.Index(arg, "."); i >= 0 {
		arg = arg[:i]
	}
	return arg
}

func plainIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isIdentByte(s[i]) {
			return false
		}
	}
	return true
}

func stripHash(line string) string {
	if i := strings.Index(line, "#"); i >= 0 {
		return line[:i]
	}
	return line
}

func parseGORM(file, src string) []draft {
	lines := strings.Split(src, "\n")
	var out []draft
	for i := 0; i < len(lines); i++ {
		name, ok := structAt(lines[i])
		if !ok {
			continue
		}
		body, next := goStruct(lines, i)
		if strings.Contains(body, "gorm:") || strings.Contains(body, "gorm.Model") {
			out = append(out, gormDraft(file, name, i+1, body))
		}
		if next > i {
			i = next
		}
	}
	return out
}

func structAt(line string) (string, bool) {
	f := strings.Fields(line)
	if len(f) < 3 || f[0] != "type" || f[2] != "struct" {
		return "", false
	}
	return f[1], true
}

func goStruct(lines []string, start int) (string, int) {
	depth := 0
	var b strings.Builder
	for i := start; i < len(lines); i++ {
		depth += strings.Count(lines[i], "{") - strings.Count(lines[i], "}")
		if i > start {
			b.WriteString(lines[i])
			b.WriteByte('\n')
		}
		if i > start && depth <= 0 {
			return b.String(), i
		}
	}
	return b.String(), len(lines) - 1
}

func gormDraft(file, name string, line int, body string) draft {
	d := draft{name: name, file: file, line: line}
	if strings.Contains(body, "gorm.Model") {
		d.columns = append(d.columns,
			Column{Name: "id", Role: "pk", Line: line},
			Column{Name: "created_at", Role: "timestamp", Line: line},
			Column{Name: "updated_at", Role: "timestamp", Line: line},
			Column{Name: "deleted_at", Role: "timestamp", Line: line},
		)
	}
	for i, lineText := range strings.Split(body, "\n") {
		addGormField(&d, lineText, line+1+i)
	}
	return d
}

func addGormField(d *draft, line string, n int) {
	f := strings.Fields(line)
	if skipGormLine(f, line) {
		return
	}
	name, typ := f[0], f[1]
	tag := tagOf(line)
	if addGormRel(d, name, typ, tag) {
		return
	}
	if !basicGo(baseType(typ)) && !strings.Contains(tag, "column") {
		return
	}
	d.columns = append(d.columns, Column{Name: name, Role: gormRole(name, tag), Line: n})
}

func skipGormLine(f []string, line string) bool {
	if len(f) < 2 || f[0] == "type" || strings.HasPrefix(f[0], "//") || f[0] == "}" {
		return true
	}
	return strings.Contains(line, "gorm.Model")
}

func addGormRel(d *draft, name, typ, tag string) bool {
	if strings.Contains(tag, "many2many") {
		d.rels = append(d.rels, Relationship{Name: name, Target: baseType(typ), Type: "ManyToMany"})
		return true
	}
	if strings.Contains(tag, "foreignKey") || strings.Contains(tag, "references") {
		d.rels = append(d.rels, Relationship{Name: name, Target: baseType(typ), Type: "ForeignKey"})
		return true
	}
	return false
}

func gormRole(name, tag string) string {
	low := toSnake(name)
	switch {
	case strings.Contains(tag, "primaryKey") || low == "id":
		return "pk"
	case strings.HasSuffix(low, "_id") && low != "id":
		return "fk"
	case isTimestamp(low):
		return "timestamp"
	case low == "data":
		return "data"
	default:
		return "feature"
	}
}

func basicGo(typ string) bool {
	switch typ {
	case "string", "bool", "byte", "rune", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64",
		"Time", "JSON", "JSONB", "Bytes", "Decimal":
		return true
	default:
		return false
	}
}

func baseType(typ string) string {
	typ = strings.TrimPrefix(typ, "*")
	typ = strings.TrimPrefix(typ, "[]")
	if i := strings.LastIndex(typ, "."); i >= 0 {
		typ = typ[i+1:]
	}
	return typ
}

func tagOf(line string) string {
	i := strings.Index(line, "`")
	if i < 0 {
		return ""
	}
	j := strings.Index(line[i+1:], "`")
	if j < 0 {
		return line[i+1:]
	}
	return line[i+1 : i+1+j]
}
