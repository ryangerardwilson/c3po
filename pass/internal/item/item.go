// Package item builds and changes 1Password item JSON.
package item

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

// NormalizeCategory maps api-credential or login to the op category.
func NormalizeCategory(raw string) (string, error) {
	switch strings.ToLower(strings.ReplaceAll(raw, "_", "-")) {
	case "api-credential":
		return "API_CREDENTIAL", nil
	case "login":
		return "LOGIN", nil
	default:
		return "", errors.New("Category must be api-credential or login. Use --template for another category.")
	}
}

// SecretCategory reports whether a category has one secret field.
func SecretCategory(category string) bool {
	switch strings.ToUpper(category) {
	case "API_CREDENTIAL", "LOGIN", "PASSWORD":
		return true
	default:
		return false
	}
}

// UsernameCategory reports whether --username applies to a category.
func UsernameCategory(category string) bool {
	switch strings.ToUpper(category) {
	case "API_CREDENTIAL", "LOGIN":
		return true
	default:
		return false
	}
}

// CategoryLabel is the category as printed after a create.
func CategoryLabel(category string) string {
	switch strings.ToUpper(category) {
	case "API_CREDENTIAL":
		return "API credential"
	case "LOGIN":
		return "login"
	default:
		return category
	}
}

// New returns the JSON for a new API credential or login.
func New(title, category, username, secret string) ([]byte, error) {
	fields := make([]map[string]string, 0, 2)
	if username != "" {
		field := map[string]string{"id": "username", "type": "STRING", "label": "username", "value": username}
		if category == "LOGIN" {
			field["purpose"] = "USERNAME"
		}
		fields = append(fields, field)
	}
	switch category {
	case "API_CREDENTIAL":
		fields = append(fields, map[string]string{"id": "credential", "type": "CONCEALED", "label": "credential", "value": secret})
	case "LOGIN":
		fields = append(fields, map[string]string{"id": "password", "type": "CONCEALED", "purpose": "PASSWORD", "label": "password", "value": secret})
	default:
		return nil, errors.New("unsupported category")
	}
	return Marshal(map[string]any{"title": title, "category": category, "fields": fields})
}

// TemplateMeta returns the title and category of item JSON.
func TemplateMeta(data []byte) (string, string, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil || doc == nil {
		return "", "", errors.New("Template is not a 1Password item.")
	}
	title := strings.TrimSpace(String(doc["title"]))
	category := strings.TrimSpace(String(doc["category"]))
	if title == "" {
		return "", "", errors.New("Template needs a title.")
	}
	if category == "" {
		return "", "", errors.New("Template needs a category.")
	}
	return title, category, nil
}

// CheckObject reports an error when data is not a JSON object.
func CheckObject(data []byte) error {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil || doc == nil {
		return errors.New("Template is not a 1Password item.")
	}
	return nil
}

// Marshal encodes doc without HTML escaping.
func Marshal(doc map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SetUsername replaces the username field, or adds one.
func SetUsername(doc map[string]any, username string) {
	setField(doc, func(field map[string]any) bool {
		return String(field["id"]) == "username" ||
			strings.EqualFold(String(field["label"]), "username") ||
			strings.EqualFold(String(field["purpose"]), "USERNAME")
	}, map[string]any{"id": "username", "type": "STRING", "purpose": "USERNAME", "label": "username", "value": username})
}

// SetSecret replaces the credential of an API credential, or the
// password of another category, or adds that field.
func SetSecret(doc map[string]any, category, secret string) {
	if strings.ToUpper(category) == "API_CREDENTIAL" {
		setField(doc, func(field map[string]any) bool {
			return String(field["id"]) == "credential" || strings.EqualFold(String(field["label"]), "credential")
		}, map[string]any{"id": "credential", "type": "CONCEALED", "label": "credential", "value": secret})
		return
	}
	setField(doc, func(field map[string]any) bool {
		return String(field["id"]) == "password" ||
			strings.EqualFold(String(field["label"]), "password") ||
			strings.EqualFold(String(field["purpose"]), "PASSWORD")
	}, map[string]any{"id": "password", "type": "CONCEALED", "purpose": "PASSWORD", "label": "password", "value": secret})
}

func setField(doc map[string]any, match func(map[string]any) bool, created map[string]any) {
	fields, _ := doc["fields"].([]any)
	for _, raw := range fields {
		field, ok := raw.(map[string]any)
		if ok && match(field) {
			field["value"] = created["value"]
			doc["fields"] = fields
			return
		}
	}
	doc["fields"] = append(fields, created)
}

// String returns value when it is a string.
func String(value any) string {
	text, _ := value.(string)
	return text
}
