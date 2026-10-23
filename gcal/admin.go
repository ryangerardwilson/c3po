package gcal

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ryangerardwilson/c3po/gcal/internal/config"
	"github.com/ryangerardwilson/c3po/gcal/internal/errs"
	"github.com/ryangerardwilson/c3po/gcal/internal/oauth"
	"github.com/ryangerardwilson/c3po/gcal/internal/pyfmt"
	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

const emptyConfig = "{\"defaults\":{\"timezone\":\"UTC\"},\"accounts\":{}}\n"

// openConfig opens config.json in $VISUAL, $EDITOR, or vim, creating a
// minimal file first when none exists.
func openConfig(e *env) error {
	path := e.dirs().ConfigFile()
	if _, err := os.Stat(path); err != nil {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(emptyConfig), 0o644); err != nil {
			return err
		}
	}
	editorCmd := e.getenv("VISUAL")
	if editorCmd == "" {
		editorCmd = e.getenv("EDITOR")
	}
	if editorCmd == "" {
		editorCmd = "vim"
	}
	parts, err := pyfmt.Split(editorCmd)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return errs.Usage("Editor command is empty. Set VISUAL or EDITOR.")
	}
	return editorError(e.editor(append(parts, path)), editorCmd)
}

func editorError(err error, editorCmd string) error {
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &exitErr):
		return errs.Usage(fmt.Sprintf("Editor exited with code %d", exitErr.ExitCode()))
	case errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist):
		return errs.Usage(fmt.Sprintf("Editor not found: %s. Set VISUAL or EDITOR.", editorCmd))
	default:
		return err
	}
}

// outputJSON accepts no params or exactly "output json".
func outputJSON(params []string, shape string) (bool, error) {
	switch {
	case len(params) == 0:
		return false, nil
	case len(params) == 2 && params[0] == "output" && params[1] == "json":
		return true, nil
	default:
		return false, errs.Usage(shape)
	}
}

type accountRow struct {
	preset, email, token string
}

func (r accountRow) json() map[string]any {
	return map[string]any{"preset": r.preset, "email": r.email, "token": r.token}
}

func (r accountRow) line() string {
	return fmt.Sprintf("account preset=%s email=%s token=%s", r.preset, r.email, r.token)
}

func accountRows(e *env, app config.App) []accountRow {
	dirs := e.dirs()
	rows := []accountRow{}
	for _, preset := range app.Presets() {
		acct := app.Accounts[preset]
		token := "missing"
		tokenPath := dirs.TokenFile(acct.Email)
		if ok, err := sqlxdb.Exists("gcal", "token", sqlxdb.Key(tokenPath), tokenPath); err == nil && ok {
			token = "present"
		}
		rows = append(rows, accountRow{preset: preset, email: acct.Email, token: token})
	}
	return rows
}

func rowsJSON(rows []accountRow) []any {
	out := make([]any, len(rows))
	for i, r := range rows {
		out[i] = r.json()
	}
	return out
}

func printJSON(w io.Writer, v any) {
	fmt.Fprintln(w, pyfmt.Dumps(v, 2))
}

func loadConfig(e *env) (config.App, error) {
	dirs := e.dirs()
	return config.Load(dirs.ConfigFile(), dirs)
}

func accountsList(params []string, e *env) error {
	asJSON, err := outputJSON(params, "Use: c3po gcal accounts list [output json]")
	if err != nil {
		return err
	}
	app, err := loadConfig(e)
	if err != nil {
		return err
	}
	rows := accountRows(e, app)
	if asJSON {
		printJSON(e.stdout, map[string]any{"accounts": rowsJSON(rows)})
		return nil
	}
	if len(rows) == 0 {
		fmt.Fprintln(e.stdout, "No Google Calendar accounts configured.")
	}
	for _, r := range rows {
		fmt.Fprintln(e.stdout, r.line())
	}
	return nil
}

func setupCheck(params []string, e *env) error {
	asJSON, err := outputJSON(params, "Use: c3po gcal setup check [output json]")
	if err != nil {
		return err
	}
	app, err := loadConfig(e)
	if err != nil {
		return err
	}
	rows := accountRows(e, app)
	if asJSON {
		printJSON(e.stdout, map[string]any{"config": app.Path, "timezone": app.Timezone, "accounts": rowsJSON(rows)})
		return nil
	}
	fmt.Fprintf(e.stdout, "config=%s\ntimezone=%s\n", app.Path, app.Timezone)
	for _, r := range rows {
		fmt.Fprintln(e.stdout, r.line())
	}
	return nil
}

// auth asks for a timezone, runs the browser flow, and saves the preset.
func auth(params []string, e *env) error {
	if len(params) != 1 {
		return errs.Usage("Use: c3po gcal auth <client_secret_path>")
	}
	dirs := e.dirs()
	secret := filepath.Clean(dirs.Expand(params[0]))
	if info, err := os.Stat(secret); err != nil || !info.Mode().IsRegular() {
		return errs.Usage(fmt.Sprintf("missing client secret file: %s", secret))
	}
	answer, err := prompt(e, "Timezone [UTC, +0530, -0430]: ")
	if err != nil {
		return err
	}
	if answer == "" {
		answer = "UTC"
	}
	tz, err := config.ValidateTimezone(answer, "<prompt>")
	if err != nil {
		return err
	}
	authorized, err := e.oauth().Authorize(secret, oauth.CalendarScopes)
	if err != nil {
		return err
	}
	acct, err := config.Upsert(dirs.ConfigFile(), secret, authorized.Email, tz, dirs)
	if err != nil {
		return err
	}
	fmt.Fprintf(e.stdout, "authorized\t%s\t%s\t%s\n", acct.Preset, acct.Email, tz)
	return nil
}

// prompt is Python input(): the answer without its newline, trimmed.
// End of input with nothing read is an error.
func prompt(e *env, text string) (string, error) {
	fmt.Fprint(e.stdout, text)
	line, err := e.stdin.ReadString('\n')
	if err != nil && line == "" {
		return "", errors.New("EOF when reading a line")
	}
	return strings.TrimSpace(line), nil
}
