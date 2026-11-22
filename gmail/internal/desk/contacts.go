package desk

import (
	"fmt"
	"os"
	"strings"

	"github.com/ryangerardwilson/c3po/gmail/internal/config"
	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
)

func (e Env) contacts(app *config.App, acct config.Account, params []string) (int, error) {
	if len(params) == 0 || (len(params) == 1 && params[0] == "list") {
		return e.listContacts(acct), nil
	}
	switch params[0] {
	case "add":
		if len(params) != 3 {
			return 0, errs.Usage("Use: c3po gmail <preset> contacts list|add <alias> <email>|delete <alias>|edit")
		}
		return e.addContact(app, acct, params[1], params[2])
	case "delete":
		if len(params) != 2 {
			return 0, errs.Usage("Use: c3po gmail <preset> contacts list|add <alias> <email>|delete <alias>|edit")
		}
		return e.deleteContact(app, acct, params[1])
	case "edit":
		if len(params) != 1 {
			return 0, errs.Usage("contacts edit does not accept extra args")
		}
		return e.editConfig(app.Path)
	default:
		return 0, errs.Usage("Use: c3po gmail <preset> contacts list|add <alias> <email>|delete <alias>|edit")
	}
}

func (e Env) listContacts(acct config.Account) int {
	if len(acct.Contacts) == 0 {
		fmt.Fprintln(e.Stdout, "no contacts configured")
		return 0
	}
	fmt.Fprintln(e.Stdout, "contacts:")
	names := make(map[string]string, len(acct.Contacts))
	for _, item := range acct.Contacts {
		names[item.Alias] = item.Email
	}
	for i, alias := range sortedKeys(names) {
		fmt.Fprintf(e.Stdout, "  %d. %s -> %s\n", i+1, alias, names[alias])
	}
	return 0
}

func (e Env) addContact(app *config.App, acct config.Account, alias, email string) (int, error) {
	alias = strings.ToLower(strings.TrimSpace(alias))
	email = strings.TrimSpace(email)
	if alias == "" {
		return 0, errs.Usage("contacts add requires a non-empty alias")
	}
	if !strings.Contains(email, "@") {
		email = resolveContact(email, acct.Contacts)
	}
	if !strings.Contains(email, "@") {
		return 0, errs.Usage("contacts add requires an email address")
	}
	next := acct.Contacts.Set(alias, email)
	if err := config.UpdateContacts(app.Path, acct.Preset, next); err != nil {
		return 0, err
	}
	fmt.Fprintf(e.Stdout, "contact added: %s -> %s\n", alias, email)
	return 0, nil
}

func (e Env) deleteContact(app *config.App, acct config.Account, alias string) (int, error) {
	alias = strings.ToLower(strings.TrimSpace(alias))
	if alias == "" {
		return 0, errs.Usage("contacts delete requires a non-empty alias")
	}
	if _, ok := acct.Contacts.Get(alias); !ok {
		fmt.Fprintf(e.Stdout, "contact not found: %s\n", alias)
		return 0, nil
	}
	if err := config.UpdateContacts(app.Path, acct.Preset, acct.Contacts.Delete(alias)); err != nil {
		return 0, err
	}
	fmt.Fprintf(e.Stdout, "contact deleted: %s\n", alias)
	return 0, nil
}

func (e Env) editConfig(path string) (int, error) {
	if path == "" {
		path = e.paths().ConfigPath()
	}
	parts := strings.Fields(editorCommand())
	if len(parts) == 0 {
		return 0, errs.Usage("Editor command is empty. Set VISUAL or EDITOR.")
	}
	if err := e.edit(append(parts, path)); err != nil {
		return 0, errs.Usage("Editor exited with code %s", err.Error())
	}
	return 0, nil
}

func (e Env) auth(params []string) (int, error) {
	if len(params) != 1 {
		return 0, errs.Usage("Use: c3po gmail auth <client_secret_path>")
	}
	secret := e.paths().Expand(params[0])
	st, err := os.Stat(secret)
	if err != nil || st.IsDir() {
		return 0, errs.Usage("Missing client secret file: %s", secret)
	}
	email, err := e.authorize(secret)
	if err != nil {
		return 0, err
	}
	sig, err := e.signatureFile(email)
	if err != nil {
		return 0, err
	}
	preset, err := config.UpsertAccount(e.paths().ConfigPath(), email, secret, sig)
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(e.Stdout, "authorized preset=%s email=%s signature_file=%s\n", preset, email, sig)
	return 0, nil
}

func (e Env) signatureFile(email string) (string, error) {
	def := e.paths().DefaultSignaturePath(email)
	answer, err := e.ask(fmt.Sprintf("Signature file path [%s]: ", def))
	if err != nil {
		return "", err
	}
	path := def
	if strings.TrimSpace(answer) != "" {
		path = e.paths().Expand(strings.TrimSpace(answer))
	}
	if err := os.MkdirAll(filepathDir(path), 0o700); err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err != nil {
		if err := os.WriteFile(path, []byte(email), 0o600); err != nil {
			return "", err
		}
	}
	return path, nil
}

func filepathDir(path string) string {
	i := strings.LastIndex(path, "/")
	if i <= 0 {
		return "."
	}
	return path[:i]
}
