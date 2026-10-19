// Package config reads and writes the gcal config.json.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/gcal/internal/errs"
	"github.com/ryangerardwilson/c3po/gcal/internal/paths"
	"github.com/ryangerardwilson/c3po/gcal/internal/pyfmt"
)

// Account is one numeric preset.
type Account struct {
	Preset           string
	Email            string
	ClientSecretFile string
}

// App is a loaded config.
type App struct {
	Path     string
	Accounts map[string]Account
	Timezone string
}

// Presets lists the preset keys in numeric order.
func (a App) Presets() []string {
	keys := make([]string, 0, len(a.Accounts))
	for k := range a.Accounts {
		keys = append(keys, k)
	}
	sortNumeric(keys)
	return keys
}

// Load reads and validates the config at path, creating the gcal
// directories first.
func Load(path string, dirs paths.Dirs) (App, error) {
	if err := dirs.Ensure(); err != nil {
		return App{}, err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return App{}, errs.Config(fmt.Sprintf("Config not found: %s. Run `c3po gcal auth <client_secret_path>` first.", path))
	}
	if err != nil {
		return App{}, err
	}
	root, err := decodeRoot(raw, path)
	if err != nil {
		return App{}, err
	}
	return parseRoot(root, path, dirs)
}

func decodeRoot(raw []byte, path string) (map[string]any, error) {
	var v any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, errs.Config(fmt.Sprintf("Invalid JSON in config %s: %s", path, err.Error()))
	}
	if dec.More() {
		return nil, errs.Config(fmt.Sprintf("Invalid JSON in config %s: Extra data", path))
	}
	root, ok := v.(map[string]any)
	if !ok {
		return nil, invalid(path, "root must be an object")
	}
	return root, nil
}

func invalid(path, detail string) error {
	return errs.Config(fmt.Sprintf("Invalid config at %s: %s", path, detail))
}

func parseRoot(root map[string]any, path string, dirs paths.Dirs) (App, error) {
	accountsRaw, ok := root["accounts"].(map[string]any)
	if !ok || len(accountsRaw) == 0 {
		return App{}, invalid(path, "'accounts' must be a non-empty object")
	}
	tz, err := defaultsTimezone(root["defaults"], path)
	if err != nil {
		return App{}, err
	}
	keys := make([]string, 0, len(accountsRaw))
	for k := range accountsRaw {
		if !isDigits(k) {
			return App{}, invalid(path, "preset keys must be numeric strings")
		}
		keys = append(keys, k)
	}
	sortNumeric(keys)
	app := App{Path: path, Accounts: map[string]Account{}, Timezone: tz}
	for _, preset := range keys {
		acct, err := parseAccount(preset, accountsRaw[preset], path, dirs)
		if err != nil {
			return App{}, err
		}
		app.Accounts[preset] = acct
	}
	return app, nil
}

func defaultsTimezone(raw any, path string) (string, error) {
	if raw == nil {
		return ValidateTimezone(nil, path)
	}
	defaults, ok := raw.(map[string]any)
	if !ok {
		return "", invalid(path, "'defaults' must be an object")
	}
	return ValidateTimezone(defaults["timezone"], path)
}

func parseAccount(preset string, raw any, path string, dirs paths.Dirs) (Account, error) {
	value, ok := raw.(map[string]any)
	if !ok {
		return Account{}, invalid(path, fmt.Sprintf("accounts['%s'] must be an object", preset))
	}
	email, _ := value["email"].(string)
	if strings.TrimSpace(email) == "" {
		return Account{}, invalid(path, fmt.Sprintf("accounts['%s'].email is required", preset))
	}
	secret, _ := value["client_secret_file"].(string)
	if strings.TrimSpace(secret) == "" {
		return Account{}, invalid(path, fmt.Sprintf("accounts['%s'].client_secret_file is required", preset))
	}
	secretPath := filepath.Clean(dirs.Expand(secret))
	if _, err := os.Stat(secretPath); err != nil {
		return Account{}, invalid(path, fmt.Sprintf("client_secret_file not found for preset '%s': %s", preset, secretPath))
	}
	return Account{Preset: preset, Email: paths.NormalizeEmail(email), ClientSecretFile: secretPath}, nil
}

// Get returns one preset or a config error naming the presets that exist.
func Get(app App, preset string) (Account, error) {
	if acct, ok := app.Accounts[preset]; ok {
		return acct, nil
	}
	keys := make([]string, 0, len(app.Accounts))
	for k := range app.Accounts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	available := strings.Join(keys, ", ")
	if available == "" {
		available = "none"
	}
	return Account{}, errs.Config(fmt.Sprintf("Preset `%s` not found. Available presets: %s", preset, available))
}

// Upsert saves the account for email, keeping its preset when the email
// is already configured and taking the next free preset otherwise. The
// whole file is rewritten with the given timezone.
func Upsert(path, clientSecret, email, timezone string, dirs paths.Dirs) (Account, error) {
	if err := dirs.Ensure(); err != nil {
		return Account{}, err
	}
	accounts, err := existingAccounts(path, dirs)
	if err != nil {
		return Account{}, err
	}
	secret, err := resolvePath(dirs.Expand(clientSecret))
	if err != nil {
		return Account{}, err
	}
	normalized := paths.NormalizeEmail(email)
	matched := Account{Preset: nextPreset(accounts), Email: normalized, ClientSecretFile: secret}
	for _, acct := range accounts {
		if acct.Email == normalized {
			matched.Preset = acct.Preset
			break
		}
	}
	accounts[matched.Preset] = matched
	if err := write(path, accounts, timezone); err != nil {
		return Account{}, err
	}
	return matched, nil
}

func existingAccounts(path string, dirs paths.Dirs) (map[string]Account, error) {
	if _, err := os.Stat(path); err != nil {
		return map[string]Account{}, nil
	}
	app, err := Load(path, dirs)
	if err != nil {
		return nil, err
	}
	return app.Accounts, nil
}

func resolvePath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real, nil
	}
	return abs, nil
}

func write(path string, accounts map[string]Account, timezone string) error {
	keys := make([]string, 0, len(accounts))
	for k := range accounts {
		keys = append(keys, k)
	}
	sortNumeric(keys)
	accts := pyfmt.Object{}
	for _, k := range keys {
		accts = append(accts, pyfmt.Pair{Key: k, Value: pyfmt.Object{
			{Key: "email", Value: accounts[k].Email},
			{Key: "client_secret_file", Value: accounts[k].ClientSecretFile},
		}})
	}
	payload := pyfmt.Object{
		{Key: "accounts", Value: accts},
		{Key: "defaults", Value: pyfmt.Object{{Key: "timezone", Value: timezone}}},
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(pyfmt.Dumps(payload, 2)+"\n"), 0o644)
}

func nextPreset(accounts map[string]Account) string {
	keys := make([]string, 0, len(accounts))
	for k := range accounts {
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return "1"
	}
	sortNumeric(keys)
	return increment(strings.TrimLeft(keys[len(keys)-1], "0"))
}

// increment adds one to a decimal string of any length.
func increment(n string) string {
	digits := []byte("0" + n)
	for i := len(digits) - 1; i >= 0; i-- {
		if digits[i] < '9' {
			digits[i]++
			break
		}
		digits[i] = '0'
	}
	return strings.TrimLeft(string(digits), "0")
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// IsDigits reports whether s is a non-empty run of ASCII digits.
func IsDigits(s string) bool { return isDigits(s) }

func sortNumeric(keys []string) {
	sort.SliceStable(keys, func(i, j int) bool {
		a, b := strings.TrimLeft(keys[i], "0"), strings.TrimLeft(keys[j], "0")
		if len(a) != len(b) {
			return len(a) < len(b)
		}
		return a < b
	})
}
