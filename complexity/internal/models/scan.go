package models

import (
	"path/filepath"
	"strings"
)

// Source is one file the model scanner can read.
type Source struct {
	Path     string
	Language string
	Text     string
}

// Scan finds models, scores the relationship graph, and applies conventions.
func Scan(files []Source, th Thresholds, conv Conventions) Section {
	symbols := map[string]bool{}
	var drafts []draft
	for _, f := range files {
		remember(symbols, strings.TrimSuffix(filepath.Base(f.Path), filepath.Ext(f.Path)))
		harvest(symbols, f.Text)
		drafts = append(drafts, parseSource(f)...)
	}
	return finish(drafts, th, conv, symbols)
}

func parseSource(f Source) []draft {
	switch f.Language {
	case "prisma":
		return parsePrisma(f.Path, f.Text)
	case "sql":
		return parseSQL(f.Path, f.Text)
	case "python":
		return parsePython(f.Path, f.Text)
	case "go":
		return parseGORM(f.Path, f.Text)
	case "typescript", "javascript":
		return parseTS(f.Path, f.Text)
	case "rust":
		return parseRust(f.Path, f.Text)
	default:
		return nil
	}
}

func harvest(dst map[string]bool, src string) {
	for _, line := range strings.Split(src, "\n") {
		if name := declared(strings.TrimSpace(line)); name != "" {
			remember(dst, name)
		}
	}
}

func declared(line string) string {
	for _, key := range []string{"class ", "def ", "func ", "type ", "model ", "fn ", "interface "} {
		if strings.HasPrefix(line, key) {
			rest := strings.TrimPrefix(line, key)
			if i := strings.IndexAny(rest, " ([:{"); i >= 0 {
				rest = rest[:i]
			}
			if plainIdent(rest) {
				return rest
			}
		}
	}
	return ""
}

func remember(dst map[string]bool, name string) {
	if name == "" || dst == nil {
		return
	}
	dst[strings.ToLower(name)] = true
	for _, p := range strings.Split(toSnake(name), "_") {
		if p != "" {
			dst[p] = true
		}
	}
}

func quotedArg(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] != '"' && s[i] != '\'' && s[i] != '`' {
			continue
		}
		q := s[i]
		j := strings.IndexByte(s[i+1:], q)
		if j >= 0 {
			return s[i+1 : i+1+j]
		}
	}
	return ""
}

func stripLineComment(line string) string {
	if i := strings.Index(line, "//"); i >= 0 {
		return line[:i]
	}
	return line
}
