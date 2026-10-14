package models

import "strings"

func parsePrisma(file, src string) []draft {
	lines := strings.Split(src, "\n")
	var out []draft
	for i := 0; i < len(lines); i++ {
		name, ok := prismaModel(lines[i])
		if !ok {
			continue
		}
		body, next := braceBlock(lines, i)
		out = append(out, prismaDraft(file, name, i+1, body))
		if next > i {
			i = next
		}
	}
	linkPrisma(out)
	return out
}

func prismaModel(line string) (string, bool) {
	f := strings.Fields(stripLineComment(line))
	if len(f) < 2 || f[0] != "model" {
		return "", false
	}
	return f[1], true
}

func braceBlock(lines []string, start int) ([]string, int) {
	depth := 0
	var body []string
	for i := start; i < len(lines); i++ {
		depth += strings.Count(lines[i], "{") - strings.Count(lines[i], "}")
		if i > start {
			body = append(body, lines[i])
		}
		if i > start && depth <= 0 {
			return body, i
		}
	}
	return body, len(lines) - 1
}

type pField struct {
	name    string
	typ     string
	list    bool
	rel     bool
	fields  bool
	pk      bool
	unique  bool
	line    int
	rawType string
}

func prismaDraft(file, name string, line int, body []string) draft {
	d := draft{name: name, file: file, line: line}
	d.table, d.columns = prismaColumns(body, line)
	d.rels = nil
	d.columns = d.columns
	return attachRaw(d, prismaFields(body, line))
}

func prismaColumns(body []string, line int) (string, []Column) {
	var table string
	var cols []Column
	for i, lineText := range body {
		trim := strings.TrimSpace(stripLineComment(lineText))
		if strings.HasPrefix(trim, "@@map(") {
			table = quotedArg(trim)
			continue
		}
		if skipPrisma(trim) {
			continue
		}
		f, ok := prismaField(trim, line+1+i)
		if !ok || f.rel || f.list {
			continue
		}
		cols = append(cols, Column{Name: f.name, Role: prismaRole(f), Line: f.line})
	}
	return table, cols
}

func skipPrisma(trim string) bool {
	return trim == "" || strings.HasPrefix(trim, "@@") || strings.HasPrefix(trim, "//")
}

func prismaFields(body []string, line int) []pField {
	var stash []pField
	for i, lineText := range body {
		text := strings.TrimSpace(stripLineComment(lineText))
		f, ok := prismaField(text, line+1+i)
		if ok {
			stash = append(stash, f)
		}
	}
	return stash
}

func attachRaw(d draft, fields []pField) draft {
	// stored temporarily in rel names until linkPrisma; keep scalar columns only
	for _, f := range fields {
		if f.rel || f.list {
			kind := "pending"
			if f.fields {
				kind = "ForeignKey"
				if f.unique && !f.list {
					kind = "OneToOne"
				}
			}
			d.rels = append(d.rels, Relationship{Name: f.name, Target: f.typ, Type: kind})
		}
	}
	return d
}

func prismaField(text string, line int) (pField, bool) {
	f := strings.Fields(text)
	if len(f) < 2 || strings.HasPrefix(f[0], "@") {
		return pField{}, false
	}
	raw := f[1]
	list := strings.Contains(raw, "[]")
	typ := strings.Trim(raw, "?[]")
	p := pField{name: f[0], typ: typ, list: list, line: line, rawType: raw}
	attr := text
	p.pk = strings.Contains(attr, "@id")
	p.unique = strings.Contains(attr, "@unique")
	p.rel = strings.Contains(attr, "@relation") || list
	p.fields = strings.Contains(attr, "fields:")
	if p.fields {
		p.rel = true
	}
	return p, true
}

func prismaRole(f pField) string {
	low := strings.ToLower(f.name)
	switch {
	case f.pk || low == "id":
		return "pk"
	case strings.HasSuffix(low, "id") && low != "id":
		return "fk"
	case isTimestamp(low) || strings.Contains(strings.ToLower(f.rawType), "datetime"):
		if isTimestamp(low) {
			return "timestamp"
		}
		return "feature"
	case low == "data" || strings.EqualFold(f.rawType, "json") || strings.EqualFold(f.rawType, "jsonb"):
		if low == "data" {
			return "data"
		}
		return "feature"
	default:
		return "feature"
	}
}

func linkPrisma(drafts []draft) {
	fk, list := prismaEdges(drafts)
	for i := range drafts {
		retarget(&drafts[i], fk, list)
	}
}

func prismaEdges(drafts []draft) (map[string]map[string]bool, map[string]map[string]bool) {
	fk := map[string]map[string]bool{}
	list := map[string]map[string]bool{}
	for _, d := range drafts {
		noteEdges(d, fk, list)
	}
	return fk, list
}

func noteEdges(d draft, fk, list map[string]map[string]bool) {
	for _, r := range d.rels {
		if r.Type == "ForeignKey" || r.Type == "OneToOne" {
			mark(fk, d.name, r.Target)
			continue
		}
		mark(list, d.name, r.Target)
	}
}

func retarget(d *draft, fk, list map[string]map[string]bool) {
	for j, r := range d.rels {
		if r.Type == "ForeignKey" || r.Type == "OneToOne" {
			continue
		}
		d.rels[j].Type = listKind(fk, list, d.name, r)
	}
}

func listKind(fk, list map[string]map[string]bool, self string, r Relationship) string {
	if !fk[r.Target][self] && !fk[self][r.Target] && list[r.Target][self] {
		return "ManyToMany"
	}
	return "reverse_FK"
}

func mark(m map[string]map[string]bool, from, to string) {
	if m[from] == nil {
		m[from] = map[string]bool{}
	}
	m[from][to] = true
}
