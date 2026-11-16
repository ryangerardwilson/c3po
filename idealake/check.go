package idealake

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func cleanTopic(raw string) (string, error) {
	return cleanField(raw, maxTopic, "topic")
}

func cleanIdeas(raw []string) ([]string, error) {
	out := make([]string, len(raw))
	for i, item := range raw {
		text, err := cleanField(item, maxIdea, fmt.Sprintf("idea %d", i+1))
		if err != nil {
			return nil, err
		}
		out[i] = text
	}
	return out, nil
}

func cleanField(raw string, limit int, label string) (string, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", fmt.Errorf("idealake: %s is empty", label)
	}
	if strings.ContainsAny(text, "\r\n") {
		return "", fmt.Errorf("idealake: %s must be one line", label)
	}
	n := utf8.RuneCountInString(text)
	if n > limit {
		return "", fmt.Errorf("idealake: %s is %d characters, the limit is %d", label, n, limit)
	}
	return text, nil
}
