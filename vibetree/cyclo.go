package vibetree

import (
	"path/filepath"
	"strings"

	fileparse "github.com/ryangerardwilson/c3po/vibetree/internal/parse"
)

// cyclomaticFile sums the cyclomatic complexity of every function in src.
// The file is the module. A directory sum is the caller's job.
// ok is false when the file has no scored function.
func cyclomaticFile(path string, src []byte) (int, bool) {
	lang := language(path)
	if !scoredLang(lang) {
		return 0, false
	}
	fns, _ := fileparse.File(path, lang, src)
	sum := 0
	for _, fn := range fns {
		sum += fn.Cyclomatic
	}
	return sum, sum > 0
}

func language(path string) string {
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

func scoredLang(lang string) bool {
	switch lang {
	case "go", "python", "rust", "typescript", "javascript":
		return true
	default:
		return false
	}
}
