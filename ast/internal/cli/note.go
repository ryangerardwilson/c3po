package cli

import (
	"fmt"
	"path/filepath"

	"github.com/ryangerardwilson/c3po/ast/internal/files"
)

func (r Runner) notedSymbol(results []files.ApplyResult) error {
	for _, res := range results {
		_, _ = fmt.Fprintf(r.Out, "ok  %s  %s  formatter %s\n", res.Hash, res.Path, formatterName(res.Formatter))
	}
	return nil
}

func (r Runner) notedHash(root, abs, hash string) error {
	rel, _ := filepath.Rel(root, abs)
	_, _ = fmt.Fprintf(r.Out, "ok  %s  %s\n", hash, filepath.ToSlash(rel))
	return nil
}
