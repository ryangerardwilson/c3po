package pyfmt

import (
	"errors"
	"strings"
)

// Split breaks a command line into words like Python shlex.split in
// POSIX mode: single quotes are literal, double quotes allow backslash
// escapes of \ " $ and `, and a bare backslash escapes the next rune.
func Split(s string) ([]string, error) {
	sp := splitter{}
	for _, r := range s {
		sp.feed(r)
	}
	return sp.finish()
}

type splitter struct {
	words  []string
	cur    strings.Builder
	inWord bool
	quote  rune
	escape bool
	escQ   rune
}

func (sp *splitter) feed(r rune) {
	switch {
	case sp.escape:
		sp.escaped(r)
	case sp.quote != 0:
		sp.quoted(r)
	case r == '\\':
		sp.escape, sp.escQ, sp.inWord = true, 0, true
	case r == '\'' || r == '"':
		sp.quote, sp.inWord = r, true
	case r == ' ' || r == '\t' || r == '\n' || r == '\r':
		sp.flush()
	default:
		sp.cur.WriteRune(r)
		sp.inWord = true
	}
}

func (sp *splitter) escaped(r rune) {
	sp.escape = false
	if sp.escQ == '"' && !strings.ContainsRune("\\\"$`\n", r) {
		sp.cur.WriteRune('\\')
	}
	sp.cur.WriteRune(r)
}

func (sp *splitter) quoted(r rune) {
	switch {
	case r == sp.quote:
		sp.quote = 0
	case r == '\\' && sp.quote == '"':
		sp.escape, sp.escQ = true, '"'
	default:
		sp.cur.WriteRune(r)
	}
}

func (sp *splitter) flush() {
	if sp.inWord {
		sp.words = append(sp.words, sp.cur.String())
	}
	sp.cur.Reset()
	sp.inWord = false
}

func (sp *splitter) finish() ([]string, error) {
	if sp.quote != 0 {
		return nil, errors.New("No closing quotation")
	}
	if sp.escape {
		return nil, errors.New("No escaped character")
	}
	sp.flush()
	return sp.words, nil
}
