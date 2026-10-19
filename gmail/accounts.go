package gmail

import (
	"fmt"
	"os"
	"sort"

	"github.com/ryangerardwilson/c3po/gmail/internal/config"
	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
	"github.com/ryangerardwilson/c3po/gmail/internal/pyjson"
)

func (e Env) accounts(params []string) (int, error) {
	if len(params) == 0 || params[0] != "list" {
		return 0, errs.Usage("Use: c3po gmail accounts list [output json]")
	}
	jsonOut, err := wantJSON(params[1:], "Use: c3po gmail accounts list [output json]")
	if err != nil {
		return 0, err
	}
	app, err := e.load()
	if err != nil {
		return 0, err
	}
	rows := e.accountRows(app)
	if jsonOut {
		e.printJSON(map[string]any{"accounts": rows})
		return 0, nil
	}
	if len(rows) == 0 {
		fmt.Fprintln(e.Stdout, "No Gmail accounts configured.")
		return 0, nil
	}
	for _, row := range rows {
		fmt.Fprintf(e.Stdout, "account preset=%s email=%s token=%s signature=%s contacts=%d\n",
			row["preset"], row["email"], row["token"], row["signature"], row["contacts"])
	}
	return 0, nil
}

func (e Env) setup(params []string) (int, error) {
	if len(params) == 0 || params[0] != "check" {
		return 0, errs.Usage("Use: c3po gmail setup check [output json]")
	}
	jsonOut, err := wantJSON(params[1:], "Use: c3po gmail setup check [output json]")
	if err != nil {
		return 0, err
	}
	app, err := e.load()
	if err != nil {
		return 0, err
	}
	rows := e.accountRows(app)
	payload := map[string]any{
		"config":   app.Path,
		"accounts": rows,
		"defaults": map[string]any{"list_limit": app.DefaultListLimit, "timezone_offset": app.TimezoneOffset},
		"timer":    e.timerState(),
	}
	if jsonOut {
		e.printJSON(payload)
		return 0, nil
	}
	fmt.Fprintf(e.Stdout, "config=%s\n", app.Path)
	fmt.Fprintf(e.Stdout, "default_list_limit=%d\n", app.DefaultListLimit)
	fmt.Fprintf(e.Stdout, "timezone_offset=%s\n", app.TimezoneOffset)
	fmt.Fprintf(e.Stdout, "timer=%s\n", payload["timer"])
	for _, row := range rows {
		fmt.Fprintf(e.Stdout, "account preset=%s email=%s token=%s signature=%s\n",
			row["preset"], row["email"], row["token"], row["signature"])
	}
	return 0, nil
}

func (e Env) accountRows(app *config.App) []map[string]any {
	presets := app.SortedPresets()
	rows := make([]map[string]any, 0, len(presets))
	for _, preset := range presets {
		acct := app.Accounts[preset]
		sig := "missing"
		if st, err := os.Stat(acct.SignatureFile); err == nil && !st.IsDir() {
			sig = "present"
		}
		token := "missing"
		if _, err := os.Stat(e.paths().TokenFile(acct.Email)); err == nil {
			token = "present"
		}
		rows = append(rows, map[string]any{
			"preset": preset, "email": acct.Email, "token": token, "signature": sig,
			"contacts": len(acct.Contacts), "spam_senders": len(acct.SpamSenders), "spam_excludes": len(acct.SpamExcludes),
		})
	}
	return rows
}

func wantJSON(params []string, shape string) (bool, error) {
	if len(params) == 0 {
		return false, nil
	}
	if len(params) == 2 && params[0] == "output" && params[1] == "json" {
		return true, nil
	}
	return false, errs.Usage("%s", shape)
}

func (e Env) printJSON(v any) {
	fmt.Fprintln(e.Stdout, pyjson.Dumps(v, 2, true))
}

func splitOutput(params []string, shape string) ([]string, bool, error) {
	if len(params) >= 2 && params[len(params)-2] == "output" && params[len(params)-1] == "json" {
		return params[:len(params)-2], true, nil
	}
	if containsWord(params, "output") {
		return nil, false, errs.Usage("%s", shape)
	}
	return params, false, nil
}

func containsWord(params []string, word string) bool {
	for _, p := range params {
		if p == word {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
