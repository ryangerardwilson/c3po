package cli

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/ast/internal/ccgate"
	"github.com/ryangerardwilson/c3po/ast/internal/config"
	"github.com/ryangerardwilson/c3po/ast/internal/files"
)

// ccRefused is a write that raised a function from below max_cc to max_cc
// or above, added one at or above it, or could not be scored. Run prints
// its lines on stdout, not an Error line.
type ccRefused struct{ lines []string }

func (e ccRefused) Error() string { return strings.Join(e.lines, "\n") }

// ccMax is max_cc, or the default when it is not positive.
func ccMax(maxCC int) int {
	if maxCC <= 0 {
		return config.DefaultMaxCC
	}
	return maxCC
}

// ccGate scores one plain patch. It fills notes with the cc lines to print
// after the ok line, or refuses the write.
func ccGate(rel string, maxCC int, notes *[]string) func(old, new []byte) error {
	return func(old, new []byte) error {
		lines, refused := ccgate.Lines(rel, maxCC, old, new)
		if len(refused) > 0 {
			return ccRefused{refused}
		}
		*notes = lines
		return nil
	}
}

// ccCheck is the rule for a symbol patch, a create, an append, and apply,
// which print no cc line. Refused lines sort by path, then name.
func ccCheck(maxCC int) func([]files.Written) error {
	return func(writes []files.Written) error {
		sorted := append([]files.Written(nil), writes...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Rel < sorted[j].Rel })
		var refused []string
		for _, w := range sorted {
			_, lines := ccgate.Lines(w.Rel, maxCC, w.Old, w.New)
			refused = append(refused, lines...)
		}
		if len(refused) > 0 {
			return ccRefused{refused}
		}
		return nil
	}
}

// refuseAppend scores the file after content is appended. A hash mismatch
// returns nil so AppendExact reports it and prints no refused line.
func refuseAppend(wsRoot, abs, expectHash string, maxCC int, content []byte) error {
	cur, err := files.HashFile(abs)
	if err != nil {
		return err
	}
	if !strings.EqualFold(cur, expectHash) {
		return nil
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(wsRoot, abs)
	if err != nil {
		return err
	}
	combined := make([]byte, 0, len(data)+len(content))
	combined = append(combined, data...)
	combined = append(combined, content...)
	return ccCheck(maxCC)([]files.Written{{
		Rel: filepath.ToSlash(rel), Old: data, New: combined,
	}})
}
