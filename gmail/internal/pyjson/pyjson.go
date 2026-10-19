// Package pyjson reads JSON keeping object key order and writes it the way
// Python's json.dumps does (ensure_ascii, ", " and ": " separators).
package pyjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Object is a JSON object that keeps insertion order like a Python dict.
type Object struct {
	keys []string
	vals map[string]any
}

// NewObject returns an empty ordered object.
func NewObject() *Object {
	return &Object{vals: map[string]any{}}
}

// Get returns the value for key.
func (o *Object) Get(key string) (any, bool) {
	v, ok := o.vals[key]
	return v, ok
}

// Set replaces a value in place or appends a new key.
func (o *Object) Set(key string, v any) {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = v
}

// Delete removes key.
func (o *Object) Delete(key string) {
	if _, ok := o.vals[key]; !ok {
		return
	}
	delete(o.vals, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			return
		}
	}
}

// Keys returns the keys in order.
func (o *Object) Keys() []string {
	return append([]string(nil), o.keys...)
}

// Len is the number of keys.
func (o *Object) Len() int { return len(o.keys) }

// Parse decodes JSON into *Object, []any, string, json.Number, bool, or nil.
func Parse(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := parseValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("extra data after JSON value")
	}
	return v, nil
}

func parseValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		if t == '{' {
			return parseObject(dec)
		}
		if t == '[' {
			return parseArray(dec)
		}
		return nil, errors.New("unexpected delimiter")
	default:
		return t, nil
	}
}

func parseObject(dec *json.Decoder) (any, error) {
	obj := NewObject()
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, _ := tok.(string)
		v, err := parseValue(dec)
		if err != nil {
			return nil, err
		}
		obj.Set(key, v)
	}
	_, err := dec.Token()
	return obj, err
}

func parseArray(dec *json.Decoder) (any, error) {
	out := []any{}
	for dec.More() {
		v, err := parseValue(dec)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	_, err := dec.Token()
	return out, err
}

// Dumps renders v like Python json.dumps. indent < 0 is the compact
// single-line form; sortKeys sorts object keys.
func Dumps(v any, indent int, sortKeys bool) string {
	w := &writer{indent: indent, sort: sortKeys}
	w.value(v, 0)
	return w.b.String()
}

type writer struct {
	b      strings.Builder
	indent int
	sort   bool
}

func (w *writer) value(v any, depth int) {
	switch t := v.(type) {
	case nil:
		w.b.WriteString("null")
	case bool:
		w.b.WriteString(strconv.FormatBool(t))
	case string:
		w.b.WriteString(Quote(t))
	case json.Number:
		w.b.WriteString(number(t))
	case int:
		w.b.WriteString(strconv.Itoa(t))
	case float64:
		w.b.WriteString(pyFloat(t))
	case *Object:
		w.object(t.keys, func(k string) any { return t.vals[k] }, depth)
	case map[string]any:
		w.object(mapKeys(t), func(k string) any { return t[k] }, depth)
	case map[string]string:
		w.object(mapKeysS(t), func(k string) any { return t[k] }, depth)
	case []any:
		w.array(len(t), func(i int) any { return t[i] }, depth)
	case []string:
		w.array(len(t), func(i int) any { return t[i] }, depth)
	case []map[string]any:
		w.array(len(t), func(i int) any { return t[i] }, depth)
	default:
		w.b.WriteString("null")
	}
}

func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func mapKeysS(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (w *writer) object(keys []string, get func(string) any, depth int) {
	if len(keys) == 0 {
		w.b.WriteString("{}")
		return
	}
	if w.sort {
		keys = append([]string(nil), keys...)
		sort.Strings(keys)
	}
	w.b.WriteByte('{')
	for i, k := range keys {
		w.sep(i, depth+1)
		w.b.WriteString(Quote(k))
		w.b.WriteString(": ")
		w.value(get(k), depth+1)
	}
	w.close(depth, '}')
}

func (w *writer) array(n int, get func(int) any, depth int) {
	if n == 0 {
		w.b.WriteString("[]")
		return
	}
	w.b.WriteByte('[')
	for i := 0; i < n; i++ {
		w.sep(i, depth+1)
		w.value(get(i), depth+1)
	}
	w.close(depth, ']')
}

func (w *writer) sep(i, depth int) {
	if w.indent < 0 {
		if i > 0 {
			w.b.WriteString(", ")
		}
		return
	}
	if i > 0 {
		w.b.WriteByte(',')
	}
	w.b.WriteByte('\n')
	w.b.WriteString(strings.Repeat(" ", w.indent*depth))
}

func (w *writer) close(depth int, c byte) {
	if w.indent >= 0 {
		w.b.WriteByte('\n')
		w.b.WriteString(strings.Repeat(" ", w.indent*depth))
	}
	w.b.WriteByte(c)
}

// Quote renders s as a Python ensure_ascii JSON string.
func Quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		quoteRune(&b, r)
	}
	b.WriteByte('"')
	return b.String()
}

var shortEscapes = map[rune]string{
	'"': `\"`, '\\': `\\`, '\n': `\n`, '\r': `\r`, '\t': `\t`, '\b': `\b`, '\f': `\f`,
}

func quoteRune(b *strings.Builder, r rune) {
	if esc, ok := shortEscapes[r]; ok {
		b.WriteString(esc)
		return
	}
	if r >= 0x20 && r <= 0x7e {
		b.WriteRune(r)
		return
	}
	if r > 0xffff {
		r -= 0x10000
		writeU(b, 0xd800|((r>>10)&0x3ff))
		writeU(b, 0xdc00|(r&0x3ff))
		return
	}
	writeU(b, r)
}

func writeU(b *strings.Builder, r rune) {
	h := strconv.FormatInt(int64(r), 16)
	b.WriteString(`\u` + strings.Repeat("0", 4-len(h)) + h)
}

func number(n json.Number) string {
	s := string(n)
	if !strings.ContainsAny(s, ".eE") {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return strconv.FormatInt(i, 10)
		}
		return s
	}
	f, err := n.Float64()
	if err != nil {
		return s
	}
	return pyFloat(f)
}

func pyFloat(f float64) string {
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	if math.IsNaN(f) {
		return "NaN"
	}
	e := strconv.FormatFloat(f, 'e', -1, 64)
	exp, _ := strconv.Atoi(e[strings.IndexByte(e, 'e')+1:])
	if exp < -4 || exp >= 16 {
		return pyExp(e)
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

func pyExp(e string) string {
	i := strings.IndexByte(e, 'e')
	mant, exp := e[:i], e[i+1:]
	sign := exp[:1]
	digits := strings.TrimLeft(exp[1:], "0")
	if len(digits) < 2 {
		digits = strings.Repeat("0", 2-len(digits)) + digits
	}
	return mant + "e" + sign + digits
}
