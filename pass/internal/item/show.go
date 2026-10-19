package item

import (
	"fmt"
	"io"
	"strings"
)

// WriteShow prints the title, id, category, and field labels of doc.
// A username is printed. A concealed value or a note is not. title, id,
// and category stand in when doc lacks them.
func WriteShow(out io.Writer, doc map[string]any, title, id, category string) {
	_, _ = fmt.Fprintf(out, "Title: %s\nID: %s\nCategory: %s\n",
		or(String(doc["title"]), title), or(String(doc["id"]), id), or(String(doc["category"]), category))
	fields, _ := doc["fields"].([]any)
	for _, raw := range fields {
		field, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if text, ok := fieldLine(field); ok {
			_, _ = fmt.Fprintln(out, text)
		}
	}
}

func fieldLine(field map[string]any) (string, bool) {
	label := or(String(field["label"]), String(field["id"]))
	if label == "" {
		return "", false
	}
	kind := or(String(field["type"]), "FIELD")
	if showsValue(field, kind) {
		return fmt.Sprintf("%s (%s): %s", label, kind, String(field["value"])), true
	}
	return label + " (" + kind + ")", true
}

// showsValue is true for a username field that is not concealed and
// has a value.
func showsValue(field map[string]any, kind string) bool {
	username := String(field["id"]) == "username" || strings.EqualFold(String(field["purpose"]), "USERNAME")
	if !username || strings.EqualFold(kind, "CONCEALED") {
		return false
	}
	return String(field["value"]) != ""
}

func or(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
