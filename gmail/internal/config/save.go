package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
	"github.com/ryangerardwilson/c3po/gmail/internal/pyjson"
)

// UpdateSpamSenders replaces one preset's spam_senders and keeps other keys.
func UpdateSpamSenders(path, preset string, senders []string) error {
	return updateAccount(path, preset, func(acct *pyjson.Object) {
		acct.Set("spam_senders", NormalizeSpamSenderList(senders))
	})
}

// UpdateSpamExcludes replaces one preset's spam_excludes.
func UpdateSpamExcludes(path, preset string, excludes []string) error {
	return updateAccount(path, preset, func(acct *pyjson.Object) {
		acct.Set("spam_excludes", NormalizeSenderList(excludes))
	})
}

// UpdateContacts replaces one preset's contacts object.
func UpdateContacts(path, preset string, contacts Contacts) error {
	return updateAccount(path, preset, func(acct *pyjson.Object) {
		acct.Set("contacts", contacts.object())
	})
}

// UpsertAccount writes the authorized email onto the matching preset, or
// the next numeric preset, and creates a config file when path is new.
func UpsertAccount(path, email, secret, signature string) (string, error) {
	root, err := loadOrEmpty(path)
	if err != nil {
		return "", err
	}
	accounts := ensureAccounts(root)
	email = NormalizeEmail(email)
	preset := presetForEmail(accounts, email)
	raw, _ := accounts.Get(preset)
	acct, _ := raw.(*pyjson.Object)
	if acct == nil {
		acct = pyjson.NewObject()
	}
	acct.Set("email", email)
	acct.Set("client_secret_file", secret)
	acct.Set("signature_file", signature)
	if _, ok := acct.Get("spam_senders"); !ok {
		acct.Set("spam_senders", []string{})
	}
	if _, ok := acct.Get("spam_excludes"); !ok {
		acct.Set("spam_excludes", []string{})
	}
	if _, ok := acct.Get("contacts"); !ok {
		acct.Set("contacts", pyjson.NewObject())
	}
	accounts.Set(preset, acct)
	if err := writeRoot(path, root); err != nil {
		return "", err
	}
	return preset, nil
}

func updateAccount(path, preset string, edit func(*pyjson.Object)) error {
	root, err := readRoot(path)
	if err != nil {
		return err
	}
	accountsRaw, _ := root.Get("accounts")
	accounts, ok := accountsRaw.(*pyjson.Object)
	if !ok {
		return errs.Config("Invalid config at %s: 'accounts' must be an object", path)
	}
	raw, _ := accounts.Get(preset)
	acct, ok := raw.(*pyjson.Object)
	if !ok {
		return errs.Config("Invalid config at %s: preset '%s' not found in accounts", path, preset)
	}
	edit(acct)
	return writeRoot(path, root)
}

func loadOrEmpty(path string) (*pyjson.Object, error) {
	if !exists(path) {
		root := pyjson.NewObject()
		root.Set("accounts", pyjson.NewObject())
		defaults := pyjson.NewObject()
		defaults.Set("list_limit", 10)
		defaults.Set("timezone_offset", "+05:30")
		root.Set("defaults", defaults)
		return root, nil
	}
	return readRoot(path)
}

func ensureAccounts(root *pyjson.Object) *pyjson.Object {
	raw, _ := root.Get("accounts")
	accounts, ok := raw.(*pyjson.Object)
	if ok {
		return accounts
	}
	accounts = pyjson.NewObject()
	root.Set("accounts", accounts)
	return accounts
}

func presetForEmail(accounts *pyjson.Object, email string) string {
	for _, preset := range accounts.Keys() {
		raw, _ := accounts.Get(preset)
		acct, ok := raw.(*pyjson.Object)
		if !ok {
			continue
		}
		existing, _ := acct.Get("email")
		if s, ok := existing.(string); ok && NormalizeEmail(s) == email {
			return preset
		}
	}
	next := 1
	for _, preset := range accounts.Keys() {
		if !IsDigits(preset) {
			continue
		}
		n, err := strconv.Atoi(preset)
		if err == nil && n >= next {
			next = n + 1
		}
	}
	return strconv.Itoa(next)
}

func writeRoot(path string, root *pyjson.Object) error {
	if err := os.MkdirAll(dirOf(path), 0o700); err != nil {
		return err
	}
	mode := os.FileMode(0o600)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	body := pyjson.Dumps(root, 2, false) + "\n"
	return os.WriteFile(path, []byte(body), mode)
}

func dirOf(path string) string {
	i := strings.LastIndex(path, "/")
	if i <= 0 {
		return "."
	}
	return path[:i]
}
