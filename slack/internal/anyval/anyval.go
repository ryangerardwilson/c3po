// Package anyval reads the loosely typed values in Slack's JSON objects.
// The core and its adapters share it so a value reads the same everywhere.
package anyval

import (
	"fmt"
	"strconv"
	"strings"
)

// Map returns value as an object, or an empty object.
func Map(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

// List returns value as a list, or nil.
func List(value any) []any {
	if typed, ok := value.([]any); ok {
		return typed
	}
	return nil
}

// Str returns value as text. A whole float prints without a fraction.
func Str(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

// Int returns value as an int, or fallback.
func Int(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

// Bool returns value as a bool.
func Bool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true" || typed == "1"
	case float64:
		return typed != 0
	case int:
		return typed != 0
	default:
		return false
	}
}

// FirstNonEmpty returns the first value that is neither blank nor "-", trimmed.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" && strings.TrimSpace(value) != "-" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// TSFloat returns a Slack timestamp as seconds, or 0.
func TSFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed
}

// MaxFloat returns the largest value, never below 0.
func MaxFloat(values ...float64) float64 {
	out := 0.0
	for _, value := range values {
		if value > out {
			out = value
		}
	}
	return out
}

// MaxInt returns the larger of a and b.
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
