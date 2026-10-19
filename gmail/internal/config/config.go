package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
	"github.com/ryangerardwilson/c3po/gmail/internal/pyjson"
)

// Contact is one alias in a preset's contacts, in config order.
type Contact struct {
	Alias string
	Email string
}

// Contacts keeps aliases in insertion order like a Python dict.
type Contacts []Contact

// Get returns the email for alias.
func (c Contacts) Get(alias string) (string, bool) {
	for _, item := range c {
		if item.Alias == alias {
			return item.Email, true
		}
	}
	return "", false
}

// Set replaces alias in place or appends it.
func (c Contacts) Set(alias, email string) Contacts {
	out := append(Contacts(nil), c...)
	for i := range out {
		if out[i].Alias == alias {
			out[i].Email = email
			return out
		}
	}
	return append(out, Contact{Alias: alias, Email: email})
}

// Delete removes alias.
func (c Contacts) Delete(alias string) Contacts {
	var out Contacts
	for _, item := range c {
		if item.Alias != alias {
			out = append(out, item)
		}
	}
	return out
}

func (c Contacts) object() *pyjson.Object {
	obj := pyjson.NewObject()
	for _, item := range c {
		obj.Set(item.Alias, item.Email)
	}
	return obj
}

// Account is one validated preset.
type Account struct {
	Preset           string
	Email            string
	ClientSecretFile string
	SignatureFile    string
	SpamSenders      []string
	SpamExcludes     []string
	Contacts         Contacts
}

// App is the loaded config.
type App struct {
	Path             string
	Presets          []string
	Accounts         map[string]Account
	DefaultListLimit int
	TimezoneOffset   string
}

var tzOffsetRe = regexp.MustCompile(`^[+-](?:[01]\d|2[0-3]):[0-5]\d$`)

// Load reads and validates the config at path.
func (p Paths) Load(path string) (*App, error) {
	path = p.Expand(path)
	if !exists(path) {
		return nil, errs.Config("Config not found: %s. Create it from example_config.json.", path)
	}
	raw, err := readRoot(path)
	if err != nil {
		return nil, err
	}
	accountsRaw, _ := raw.Get("accounts")
	accounts, ok := accountsRaw.(*pyjson.Object)
	if !ok || accounts.Len() == 0 {
		return nil, errs.Config("Invalid config at %s: 'accounts' must be a non-empty object", path)
	}
	app := &App{Path: path, Accounts: map[string]Account{}}
	for _, preset := range accounts.Keys() {
		if PyStrip(preset) == "" {
			return nil, errs.Config("Invalid config at %s: account preset keys must be strings", path)
		}
		v, _ := accounts.Get(preset)
		acct, err := p.validateAccount(preset, v, path)
		if err != nil {
			return nil, err
		}
		app.Presets = append(app.Presets, preset)
		app.Accounts[preset] = acct
	}
	if err := loadDefaults(app, raw, path); err != nil {
		return nil, err
	}
	return app, nil
}

func readRoot(path string) (*pyjson.Object, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	v, err := pyjson.Parse(data)
	if err != nil {
		return nil, errs.Config("Invalid JSON in config %s: %s", path, err.Error())
	}
	obj, ok := v.(*pyjson.Object)
	if !ok {
		return nil, errs.Config("Invalid config at %s: root must be an object", path)
	}
	return obj, nil
}

func loadDefaults(app *App, raw *pyjson.Object, path string) error {
	d, _ := raw.Get("defaults")
	if d == nil {
		d = pyjson.NewObject()
	}
	defaults, ok := d.(*pyjson.Object)
	if !ok {
		return errs.Config("Invalid config at %s: 'defaults' must be an object", path)
	}
	limit, err := listLimit(defaults, path)
	if err != nil {
		return err
	}
	tzv, _ := defaults.Get("timezone_offset")
	tz, err := ValidateTimezoneOffset(tzv, path)
	if err != nil {
		return err
	}
	app.DefaultListLimit = limit
	app.TimezoneOffset = tz
	return nil
}

func listLimit(defaults *pyjson.Object, path string) (int, error) {
	v, ok := defaults.Get("list_limit")
	if !ok {
		return 10, nil
	}
	n := -1
	switch t := v.(type) {
	case json.Number:
		if i, err := strconv.Atoi(string(t)); err == nil {
			n = i
		}
	case bool:
		if t {
			n = 1
		}
	}
	if n <= 0 {
		return 0, errs.Config("Invalid config at %s: defaults.list_limit must be a positive integer", path)
	}
	return n, nil
}

// ValidateTimezoneOffset checks defaults.timezone_offset.
func ValidateTimezoneOffset(v any, path string) (string, error) {
	if v == nil {
		return "+05:30", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", errs.Config("Invalid config at %s: defaults.timezone_offset must be a string like '+05:30'", path)
	}
	s = PyStrip(s)
	if !tzOffsetRe.MatchString(s) {
		return "", errs.Config("Invalid config at %s: defaults.timezone_offset must match ±HH:MM (example: '+05:30')", path)
	}
	return s, nil
}

func requiredString(obj *pyjson.Object, key string) (string, bool) {
	v, _ := obj.Get(key)
	s, ok := v.(string)
	if !ok || PyStrip(s) == "" {
		return "", false
	}
	return s, true
}

func (p Paths) validateAccount(preset string, raw any, path string) (Account, error) {
	obj, ok := raw.(*pyjson.Object)
	if !ok {
		return Account{}, errs.Config("Invalid config at %s: accounts['%s'] must be an object", path, preset)
	}
	fields := [3]string{}
	for i, key := range []string{"email", "client_secret_file", "signature_file"} {
		s, ok := requiredString(obj, key)
		if !ok {
			return Account{}, errs.Config("Invalid config at %s: accounts['%s'].%s is required", path, preset, key)
		}
		fields[i] = s
	}
	secret := p.Expand(fields[1])
	if !exists(secret) {
		return Account{}, errs.Config("Invalid config at %s: client_secret_file not found for preset '%s': %s", path, preset, secret)
	}
	signature := p.Expand(fields[2])
	if !exists(signature) {
		return Account{}, errs.Config("Invalid config at %s: signature_file not found for preset '%s': %s", path, preset, signature)
	}
	spam, _ := obj.Get("spam_senders")
	excludes, _ := obj.Get("spam_excludes")
	contacts, _ := obj.Get("contacts")
	return Account{
		Preset:           preset,
		Email:            NormalizeEmail(fields[0]),
		ClientSecretFile: secret,
		SignatureFile:    signature,
		SpamSenders:      NormalizeSpamSenderList(spam),
		SpamExcludes:     NormalizeSenderList(excludes),
		Contacts:         NormalizeContacts(contacts),
	}, nil
}

// Account returns the preset or a ConfigError listing the presets.
func (a *App) Account(preset string) (Account, error) {
	acct, ok := a.Accounts[preset]
	if !ok {
		names := append([]string(nil), a.Presets...)
		sort.Strings(names)
		return Account{}, errs.Config("Preset '%s' not found in %s. Available presets: %s", preset, a.Path, strings.Join(names, ", "))
	}
	return acct, nil
}

// SortedPresets orders presets numerically, non-numeric last.
func (a *App) SortedPresets() []string {
	out := append([]string(nil), a.Presets...)
	sort.SliceStable(out, func(i, j int) bool { return presetRank(out[i]) < presetRank(out[j]) })
	return out
}

func presetRank(p string) int {
	if !IsDigits(p) {
		return 9999
	}
	n, err := strconv.Atoi(p)
	if err != nil {
		return 9999
	}
	return n
}

// IsDigits matches Python's str.isdigit for decimal digits.
func IsDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// NormalizeSenderList lowercases, trims, and de-duplicates string items.
func NormalizeSenderList(v any) []string {
	items, ok := v.([]any)
	if !ok {
		if ss, ok := v.([]string); ok {
			items = stringsToAny(ss)
		}
	}
	out := []string{}
	seen := map[string]bool{}
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			continue
		}
		s = strings.ToLower(PyStrip(s))
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func stringsToAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// NormalizeSpamSenderList is NormalizeSenderList without @gmail.com senders.
func NormalizeSpamSenderList(v any) []string {
	out := []string{}
	for _, s := range NormalizeSenderList(v) {
		if !strings.HasSuffix(s, "@gmail.com") {
			out = append(out, s)
		}
	}
	return out
}

// NormalizeContacts keeps string aliases and emails, lowercasing aliases.
func NormalizeContacts(v any) Contacts {
	var out Contacts
	switch t := v.(type) {
	case *pyjson.Object:
		for _, k := range t.Keys() {
			val, _ := t.Get(k)
			out = addContact(out, k, val)
		}
	case Contacts:
		for _, item := range t {
			out = addContact(out, item.Alias, item.Email)
		}
	}
	return out
}

func addContact(out Contacts, key string, val any) Contacts {
	email, ok := val.(string)
	if !ok {
		return out
	}
	alias := strings.ToLower(PyStrip(key))
	email = PyStrip(email)
	if alias == "" || email == "" {
		return out
	}
	return out.Set(alias, email)
}

func describe(v any) string { return fmt.Sprint(v) }
