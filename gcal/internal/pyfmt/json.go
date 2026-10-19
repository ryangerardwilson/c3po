// Package pyfmt reproduces the Python text formats the old gcal printed:
// json.dumps output, str repr, and shlex.split.
package pyfmt

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Pair is one key of an Object.
type Pair struct {
	Key   string
	Value any
}

// Object is a JSON object that keeps its key order.
type Object []Pair

// Dumps encodes v like Python json.dumps with ensure_ascii. indent < 0 is
// the compact form with ", " and ": " separators. A map[string]any has
// its keys sorted, like sort_keys=True. Supported values are nil, bool,
// string, int, float64, []string, []any, map[string]any, and Object.
func Dumps(v any, indent int) string {
	var b strings.Builder
	e := encoder{b: &b, indent: indent}
	e.value(v, 0)
	return b.String()
}

type encoder struct {
	b      *strings.Builder
	indent int
}

func (e encoder) value(v any, depth int) {
	switch x := v.(type) {
	case nil:
		e.b.WriteString("null")
	case bool:
		e.b.WriteString(strconv.FormatBool(x))
	case string:
		e.b.WriteString(Quote(x))
	case int:
		e.b.WriteString(strconv.Itoa(x))
	case float64:
		e.b.WriteString(strconv.FormatFloat(x, 'g', -1, 64))
	case []string:
		e.list(len(x), func(i int) any { return x[i] }, depth)
	case []any:
		e.list(len(x), func(i int) any { return x[i] }, depth)
	case map[string]any:
		e.object(sortedPairs(x), depth)
	case Object:
		e.object(x, depth)
	default:
		e.b.WriteString(Quote(fmt.Sprint(x)))
	}
}

func sortedPairs(m map[string]any) Object {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(Object, 0, len(keys))
	for _, k := range keys {
		out = append(out, Pair{k, m[k]})
	}
	return out
}

func (e encoder) list(n int, at func(int) any, depth int) {
	if n == 0 {
		e.b.WriteString("[]")
		return
	}
	e.b.WriteByte('[')
	for i := 0; i < n; i++ {
		e.sep(i, depth+1)
		e.value(at(i), depth+1)
	}
	e.newline(depth)
	e.b.WriteByte(']')
}

func (e encoder) object(pairs Object, depth int) {
	if len(pairs) == 0 {
		e.b.WriteString("{}")
		return
	}
	e.b.WriteByte('{')
	for i, p := range pairs {
		e.sep(i, depth+1)
		e.b.WriteString(Quote(p.Key))
		e.b.WriteString(": ")
		e.value(p.Value, depth+1)
	}
	e.newline(depth)
	e.b.WriteByte('}')
}

func (e encoder) sep(i, depth int) {
	if i > 0 {
		e.b.WriteByte(',')
		if e.indent < 0 {
			e.b.WriteByte(' ')
		}
	}
	e.newline(depth)
}

func (e encoder) newline(depth int) {
	if e.indent < 0 {
		return
	}
	e.b.WriteByte('\n')
	e.b.WriteString(strings.Repeat(" ", e.indent*depth))
}

// Quote is a JSON string with every non-ASCII rune escaped, like Python.
func Quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		writeJSONRune(&b, r)
	}
	b.WriteByte('"')
	return b.String()
}

var jsonEscapes = map[rune]string{
	'"': `\"`, '\\': `\\`, '\n': `\n`, '\r': `\r`, '\t': `\t`, '\b': `\b`, '\f': `\f`,
}

func writeJSONRune(b *strings.Builder, r rune) {
	if esc, ok := jsonEscapes[r]; ok {
		b.WriteString(esc)
		return
	}
	switch {
	case r < 0x20:
		fmt.Fprintf(b, `\u%04x`, r)
	case r < 0x80:
		b.WriteRune(r)
	case r > 0xffff:
		r -= 0x10000
		fmt.Fprintf(b, `\u%04x\u%04x`, 0xd800+(r>>10), 0xdc00+(r&0x3ff))
	default:
		fmt.Fprintf(b, `\u%04x`, r)
	}
}
