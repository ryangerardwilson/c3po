package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/internal/gocyclo"
)

// ccRefused is a plain Go patch that raised a function declaration from
// below max_cc to max_cc or above. Run prints its lines on stdout, not
// an Error line.
type ccRefused struct{ lines []string }

func (e ccRefused) Error() string { return strings.Join(e.lines, "\n") }

// ccGate scores the Go function declarations of one patch. It fills notes
// with the cc lines to print after the ok line, or refuses the write.
func ccGate(rel string, maxCC int, notes *[]string) func(old, new []byte) error {
	return func(old, new []byte) error {
		if filepath.Ext(rel) != ".go" {
			return nil
		}
		lines, refused := ccLines(rel, maxCC, old, new)
		if len(refused) > 0 {
			return ccRefused{refused}
		}
		*notes = lines
		return nil
	}
}

// ccLines compares the scores of old and new by declaration name, sorted.
// Old or new bytes that are not a clean parse give one unparsed line.
func ccLines(rel string, maxCC int, old, new []byte) (notes, refused []string) {
	before, okOld := gocyclo.File(old)
	after, okNew := gocyclo.File(new)
	if !okOld || !okNew {
		return []string{fmt.Sprintf("cc  %s  unparsed", rel)}, nil
	}
	for _, name := range unionNames(before, after) {
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
	}
	return notes, refused
}

func unionNames(a, b map[string]int) []string {
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
