// Package score scores the functions of one file the way cyclotree does.
// It is outside package cyclotree because cyclotree imports ast, and c3po
// ast scores with it.
package score

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/cyclotree/internal/parse"
)

// Function is one scored function.
type Function struct {
	Language, Name   string
	Line, Cyclomatic int
}

// Language is the language cyclotree scores path as: go, python, rust,
// typescript, or javascript. Any other file is "".
func Language(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	default:
		return ""
	}
}

// File scores the functions in src, the bytes of the slash path rel. A file
// in no scored language, a file with a NUL byte, and a function whose
// cyclomatic number is not positive give no entry. Go numbers come from
// internal/gocyclo, the counter the c3po ast patch gate uses.
func File(rel string, src []byte) []Function {
	lang := Language(rel)
	if lang == "" || bytes.IndexByte(src, 0) >= 0 {
		return nil
	}
	fns, _ := parse.File(rel, lang, src)
	var out []Function
	for _, fn := range fns {
		if fn.Cyclomatic <= 0 {
			continue
		}
		out = append(out, Function{Language: fn.Language, Name: fn.Name, Line: fn.Line, Cyclomatic: fn.Cyclomatic})
	}
	return out
}
