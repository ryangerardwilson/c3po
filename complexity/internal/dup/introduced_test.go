package dup

import "testing"

const body = `func a() {
	total := load(ctx, id)
	if total.Count > limit {
		total.Trim(limit)
	}
	write(total, path)
	notify(owner, total)
	return finish(total)
}
`

func TestIntroducedNamesTheOriginal(t *testing.T) {
	copied := "func b() {\n\tstart()\n" + body[11:]
	pool := []File{{Path: "a.go", Src: body}, {Path: "b.go", Src: copied}}
	changed := map[string]Fresh{"b.go": func(int) bool { return true }}
	bodies := map[string][]Body{"a.go": {{1, 9}}, "b.go": {{1, 10}}}
	hits := Introduced(changed, pool, bodies, 6)
	if len(hits) != 1 || hits[0].File != "b.go" || hits[0].OfFile != "a.go" || hits[0].OfLine != 2 {
		t.Fatalf("%+v", hits)
	}
}

func TestIntroducedIgnoresUnchangedCopies(t *testing.T) {
	pool := []File{{Path: "a.go", Src: body}, {Path: "b.go", Src: body}}
	changed := map[string]Fresh{"b.go": func(int) bool { return false }}
	if hits := Introduced(changed, pool, nil, 6); len(hits) != 0 {
		t.Fatalf("%+v", hits)
	}
}
