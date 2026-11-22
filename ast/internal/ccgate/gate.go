// Package ccgate refuses a write that adds a function at or above max_cc,
// or raises one from below that line, in Go, TypeScript, and Python.
package ccgate

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/gocyclo"
	"github.com/ryangerardwilson/c3po/internal/score"
)

// Gated reports whether rel is a language the write gate scores.
func Gated(rel string) bool {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".go", ".ts", ".tsx", ".py":
		return true
	default:
		return false
	}
}

// Lines compares the scores of old and new. A nil old is an absent file.
// notes are the plain-patch cc lines. refused is non-empty when the write
// must not happen. An unparsed Go file, or two functions that share a name,
// refuses with no function name.
func Lines(rel string, maxCC int, old, new []byte) (notes, refused []string) {
	if !Gated(rel) {
		return nil, nil
	}
	before, okOld := snapshot(rel, old)
	after, okNew := snapshot(rel, new)
	if !okOld || !okNew {
		return nil, []string{fmt.Sprintf("refused  %s  unparsed", rel)}
	}
	for _, name := range union(before, after) {
		notes, refused = scoreOne(rel, maxCC, name, before, after, notes, refused)
	}
	return notes, refused
}

func scoreOne(rel string, maxCC int, name string, before, after map[string]int, notes, refused []string) ([]string, []string) {
	o, inOld := before[name]
	n, inNew := after[name]
	switch {
	case !inNew:
		notes = append(notes, fmt.Sprintf("cc  %s:%s  %d -> gone", rel, name, o))
	case !inOld:
		notes = append(notes, fmt.Sprintf("cc  %s:%s  new -> %d", rel, name, n))
		if n >= maxCC {
			refused = append(refused, fmt.Sprintf("refused  %s:%s  cc new -> %d crosses %d", rel, name, n, maxCC))
		}
	case o != n:
		notes = append(notes, fmt.Sprintf("cc  %s:%s  %d -> %d", rel, name, o, n))
		if o < maxCC && n >= maxCC {
			refused = append(refused, fmt.Sprintf("refused  %s:%s  cc %d -> %d crosses %d", rel, name, o, n, maxCC))
		}
	}
	return notes, refused
}

func snapshot(rel string, src []byte) (map[string]int, bool) {
	if src == nil {
		return map[string]int{}, true
	}
	if strings.ToLower(filepath.Ext(rel)) == ".go" {
		return gocyclo.File(src)
	}
	return scored(rel, src)
}

func scored(rel string, src []byte) (map[string]int, bool) {
	fns := score.File(rel, src)
	out := make(map[string]int, len(fns))
	for _, fn := range fns {
		if _, dup := out[fn.Name]; dup {
			return nil, false
		}
		out[fn.Name] = fn.Cyclomatic
	}
	return out, true
}

func union(a, b map[string]int) []string {
	seen := make(map[string]bool, len(a)+len(b))
	var names []string
	for _, m := range []map[string]int{a, b} {
		for name := range m {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	return names
}
