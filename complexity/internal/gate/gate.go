package gate

import "github.com/ryangerardwilson/c3po/complexity/internal/metric"
import "github.com/ryangerardwilson/c3po/complexity/internal/models"

// Regressions returns functions whose scores newly exceed a threshold
// or grew while already over it. Identity is described on Match.
func Regressions(cur, base []metric.Function, th metric.Thresholds) []metric.Function {
	var out []metric.Function
	for _, p := range Match(cur, base, nil) {
		if bad := Worse(p, th); len(bad) > 0 {
			fn := p.Cur
			fn.Violations = bad
			out = append(out, fn)
		}
	}
	return out
}

// Worse lists the metrics of p that are above their passing score.
func Worse(p Pair, th metric.Thresholds) []string {
	var bad []string
	for _, name := range metric.Names {
		limit := metric.Limit(th, name)
		if limit > 0 && metric.Value(p.Cur, name) > PassAt(limit, p.Base, name) {
			bad = append(bad, name)
		}
	}
	return bad
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// ModelRegressions fails new models that violate a rule, models whose
// structural score got worse, and models that newly break a convention.
func ModelRegressions(cur, base []models.Model, th models.Thresholds) []models.Model {
	idx := map[string]models.Model{}
	for _, m := range base {
		idx[fileKey(m.File, m.Name)] = m
	}
	var out []models.Model
	for _, m := range cur {
		old, ok := idx[fileKey(m.File, m.Name)]
		if !ok {
			if len(m.Violations) > 0 {
				out = append(out, m)
			}
			continue
		}
		if modelWorse(m, old, th) {
			out = append(out, m)
		}
	}
	return out
}

func modelWorse(cur, old models.Model, th models.Thresholds) bool {
	if metricUp(th.Degree, cur.Degree, old.Degree) {
		return true
	}
	if metricUp(th.Depth, cur.Depth, old.Depth) {
		return true
	}
	if metricUp(th.M2M, cur.M2M, old.M2M) {
		return true
	}
	return ruleNew(cur, old)
}

func metricUp(limit, cur, old int) bool {
	return limit > 0 && cur > limit && cur > old
}

func ruleNew(cur, old models.Model) bool {
	if cur.Cycles && !old.Cycles {
		return true
	}
	if has(cur, "naming") && !has(old, "naming") {
		return true
	}
	return has(cur, "column_order") && !has(old, "column_order")
}

func has(m models.Model, v string) bool {
	for _, s := range m.Violations {
		if s == v {
			return true
		}
	}
	return false
}
