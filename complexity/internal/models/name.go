package models

import (
	"strings"
	"unicode"
)

func finish(drafts []draft, th Thresholds, conv Conventions, symbols map[string]bool) Section {
	models := make([]Model, 0, len(drafts))
	for _, d := range drafts {
		models = append(models, modelFrom(d, conv, symbols))
	}
	annotate(models)
	var sec Section
	sec.Summary.ModelsAnalyzed = len(models)
	for i := range models {
		applyLimits(&models[i], th)
		if th.Degree > 0 && models[i].Degree > th.Degree {
			sec.Summary.HighDegree++
		}
		if models[i].Depth > sec.Summary.MaxDepth {
			sec.Summary.MaxDepth = models[i].Depth
		}
		if models[i].Cycles {
			sec.Summary.Cycles++
		}
		sec.Summary.TotalM2M += models[i].M2M
		sec.Models = append(sec.Models, models[i])
		if len(models[i].Violations) > 0 || len(models[i].Warnings) > 0 {
			sec.Hotspots = append(sec.Hotspots, models[i])
		}
	}
	return sec
}

func modelFrom(d draft, conv Conventions, symbols map[string]bool) Model {
	m := Model{
		Name: d.name, File: d.file, Line: d.line,
		Relationships: d.rels, Columns: d.columns,
	}
	checked := d.name
	if d.table != "" {
		checked = d.table
	}
	applyName(&m, checked, conv, symbols)
	applyOrder(&m, conv)
	return m
}

func applyLimits(m *Model, th Thresholds) {
	if th.Degree > 0 && m.Degree > th.Degree {
		m.Violations = append(m.Violations, "degree")
	}
	if th.Depth > 0 && m.Depth > th.Depth {
		m.Violations = append(m.Violations, "depth")
	}
	if th.M2M > 0 && m.M2M > th.M2M {
		m.Violations = append(m.Violations, "m2m")
	}
	if m.Cycles {
		m.Violations = append(m.Violations, "cycles")
	}
}

func applyName(m *Model, checked string, conv Conventions, symbols map[string]bool) {
	if !conv.Naming {
		return
	}
	suffix := conv.LogSuffix
	if suffix == "" {
		suffix = "logs"
	}
	if ok, suggestion := nameOK(checked, suffix); !ok {
		m.Violations = append(m.Violations, "naming")
		m.Suggestions = append(m.Suggestions, "rename table/model to "+suggestion)
	}
	if conv.RequireFeatureMapping && symbols != nil {
		if feat := missingFeature(checked, suffix, symbols); feat != "" {
			m.Warnings = append(m.Warnings, "feature "+feat+" does not map to a class or module")
		}
	}
}

func applyOrder(m *Model, conv Conventions) {
	if !conv.ColumnOrder || len(m.Columns) == 0 {
		return
	}
	data := conv.DataColumnName
	if data == "" {
		data = "data"
	}
	for i := range m.Columns {
		if m.Columns[i].Role == "feature" && m.Columns[i].Name == data {
			m.Columns[i].Role = "data"
		}
	}
	if ok, suggestion := orderOK(m.Columns); !ok {
		m.Violations = append(m.Violations, "column_order")
		m.Suggestions = append(m.Suggestions, suggestion)
	}
}

func nameOK(name, suffix string) (bool, string) {
	n := toSnake(strings.Trim(name, "`\""))
	if needsLog(n) && !strings.HasSuffix(n, "_"+suffix) {
		return false, logSuggestion(n, suffix)
	}
	parts := strings.Split(n, "_")
	body := parts
	if len(parts) > 0 && parts[len(parts)-1] == suffix {
		body = parts[:len(parts)-1]
	}
	if len(body) < 2 || !snakeParts(body) {
		base := n
		if base == "" {
			base = "model"
		}
		return false, base + "_core"
	}
	return true, ""
}

func logSuggestion(n, suffix string) string {
	if strings.HasSuffix(n, "_log") {
		return strings.TrimSuffix(n, "_log") + "_" + suffix
	}
	return n + "_" + suffix
}

func needsLog(n string) bool {
	for _, p := range strings.Split(n, "_") {
		switch p {
		case "log", "logs", "audit", "event", "events":
			return true
		}
	}
	return false
}

func snakeParts(parts []string) bool {
	if len(parts) == 0 {
		return false
	}
	for _, p := range parts {
		if !snakeWord(p) {
			return false
		}
	}
	return true
}

func snakeWord(p string) bool {
	if p == "" || p[0] < 'a' || p[0] > 'z' {
		return false
	}
	for i := 0; i < len(p); i++ {
		if !snakeChar(p[i]) {
			return false
		}
	}
	return true
}

func snakeChar(c byte) bool {
	letter := c >= 'a' && c <= 'z'
	digit := c >= '0' && c <= '9'
	return letter || digit
}

func missingFeature(name, suffix string, symbols map[string]bool) string {
	n := toSnake(strings.Trim(name, "`\""))
	parts := strings.Split(n, "_")
	if len(parts) > 0 && parts[len(parts)-1] == suffix {
		parts = parts[:len(parts)-1]
	}
	if len(parts) < 2 {
		return ""
	}
	for _, feat := range parts[1:] {
		if !symbolHit(feat, symbols) {
			return feat
		}
	}
	return ""
}

func symbolHit(feat string, symbols map[string]bool) bool {
	if symbols[feat] {
		return true
	}
	if len(feat) < 3 {
		return false
	}
	for sym := range symbols {
		if len(sym) < 3 {
			continue
		}
		if strings.Contains(sym, feat) || strings.Contains(feat, sym) {
			return true
		}
	}
	return false
}

func orderOK(cols []Column) (bool, string) {
	prev := -1
	for _, c := range cols {
		r := roleRank(c.Role)
		if r < prev {
			return false, "move columns to " + reorder(cols)
		}
		prev = r
	}
	return true, ""
}

func roleRank(role string) int {
	switch role {
	case "pk":
		return 0
	case "fk":
		return 1
	case "feature":
		return 2
	case "data":
		return 3
	case "timestamp":
		return 4
	default:
		return 2
	}
}

func reorder(cols []Column) string {
	ranks := []string{"pk", "fk", "feature", "data", "timestamp"}
	var names []string
	for _, role := range ranks {
		for _, c := range cols {
			if c.Role == role {
				names = append(names, c.Name)
			}
		}
	}
	return strings.Join(names, ", ")
}

func toSnake(name string) string {
	var b strings.Builder
	prevLower := false
	for _, r := range name {
		if r == '-' || r == ' ' {
			b.WriteByte('_')
			prevLower = false
			continue
		}
		if unicode.IsUpper(r) {
			if prevLower {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			prevLower = false
			continue
		}
		b.WriteRune(unicode.ToLower(r))
		prevLower = unicode.IsLetter(r)
	}
	return b.String()
}

// Hard reports whether the model should fail a strict or structural check.
func Hard(m Model, strict bool) bool {
	for _, v := range m.Violations {
		if v == "naming" || v == "column_order" {
			if strict {
				return true
			}
			continue
		}
		return true
	}
	return false
}
