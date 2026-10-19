package pyfmt

import (
	"fmt"
	"strings"
	"unicode"
)

// Repr is Python's repr of a str: single quotes unless the text has a
// single quote and no double quote.
func Repr(s string) string {
	quote := '\''
	if strings.ContainsRune(s, '\'') && !strings.ContainsRune(s, '"') {
		quote = '"'
	}
	var b strings.Builder
	b.WriteRune(quote)
	for _, r := range s {
		writeReprRune(&b, r, quote)
	}
	b.WriteRune(quote)
	return b.String()
}

func writeReprRune(b *strings.Builder, r, quote rune) {
	switch {
	case r == quote || r == '\\':
		b.WriteByte('\\')
		b.WriteRune(r)
	case r == '\n':
		b.WriteString(`\n`)
	case r == '\r':
		b.WriteString(`\r`)
	case r == '\t':
		b.WriteString(`\t`)
	case r == ' ' || (r < 0x7f && r >= 0x20):
		b.WriteRune(r)
	case r < 0x100 && !printable(r):
		fmt.Fprintf(b, `\x%02x`, r)
	case printable(r):
		b.WriteRune(r)
	case r < 0x10000:
		fmt.Fprintf(b, `\u%04x`, r)
	default:
		fmt.Fprintf(b, `\U%08x`, r)
	}
}

func printable(r rune) bool {
	return r >= 0xa0 && r != 0xad && unicode.IsPrint(r)
}
