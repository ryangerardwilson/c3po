package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/pass/internal/op"
	"github.com/ryangerardwilson/c3po/pass/internal/vault"
)

func isGlobArg(arg string) bool {
	return vault.IsGlob(arg)
}

func (r Runner) listVault(name string, rest []string) int {
	if len(rest) != 0 {
		return r.fail("Use: c3po pass <vault> -l")
	}
	name, client, ok := r.begin(name)
	if !ok {
		return 1
	}
	items, err := client.List(name)
	if err != nil {
		return r.fail(err.Error())
	}
	lines := vault.Lines(items)
	if len(lines) == 0 {
		_, _ = fmt.Fprintf(r.Out, "No items in vault %s.\n", name)
		return 0
	}
	for _, line := range lines {
		_, _ = fmt.Fprintln(r.Out, line)
	}
	return 0
}

func (r Runner) lookup(name, search string) int {
	name = strings.TrimSpace(name)
	search = strings.TrimSpace(search)
	if name == "" || search == "" {
		return r.fail("Vault and search must not be empty.")
	}
	client, ok := r.session()
	if !ok {
		return 1
	}
	items, err := client.List(name)
	if err != nil {
		return r.fail(err.Error())
	}
	matches := vault.Find(items, search)
	switch len(matches) {
	case 0:
		return r.fail(fmt.Sprintf("No matches for %s in vault %s.", search, name))
	case 1:
		return r.copyMatch(client, matches[0])
	default:
		return r.pickMatch(client, name, search, matches)
	}
}

// pickMatch numbers the matches. A terminal chooses one. A shell fails.
func (r Runner) pickMatch(client op.Client, name, search string, matches []vault.Match) int {
	for i, match := range matches {
		_, _ = fmt.Fprintln(r.Out, match.Format(i+1))
	}
	if r.forShell() {
		return r.fail(fmt.Sprintf("More than one match for %s in vault %s.", search, name))
	}
	index, err := r.choose(len(matches))
	if err != nil {
		return r.fail(err.Error())
	}
	return r.copyMatch(client, matches[index])
}

// copyMatch copies the secret in a terminal and prints it to a shell.
func (r Runner) copyMatch(client op.Client, match vault.Match) int {
	secret, err := client.Read(match.VaultID, match.ID, match.Category)
	if err != nil {
		return r.fail(err.Error())
	}
	if r.forShell() {
		_, _ = fmt.Fprintln(r.Out, secret)
		return 0
	}
	if err := r.Clip.Copy(secret, r.Err); err != nil {
		return r.fail(err.Error())
	}
	if match.Username != "" {
		_, _ = fmt.Fprintf(r.Out, "Copied password for %s (%s)\n", match.Title, match.Username)
	} else {
		_, _ = fmt.Fprintf(r.Out, "Copied password for %s\n", match.Title)
	}
	return 0
}

func (r Runner) mapSecrets(name string, selectors []string) int {
	name = strings.TrimSpace(name)
	if name == "" {
		return r.fail("Vault must not be empty.")
	}
	client, ok := r.session()
	if !ok {
		return 1
	}
	items, err := client.List(name)
	if err != nil {
		return r.fail(err.Error())
	}
	chosen, err := vault.Select(items, selectors)
	if err != nil {
		return r.fail(err.Error())
	}
	values, keys, err := readAll(client, chosen)
	if err != nil {
		return r.fail(err.Error())
	}
	return r.emitMap(values, keys)
}

// readAll reads each secret. The first title wins when two items share it.
func readAll(client op.Client, chosen []op.Item) (map[string]string, []string, error) {
	values := map[string]string{}
	var keys []string
	for _, item := range chosen {
		title := vault.DisplayTitle(item)
		secret, err := client.Read(item.Vault.ID, item.ID, item.Category)
		if err != nil {
			return nil, nil, fmt.Errorf("No secret available for %s.", title)
		}
		if _, ok := values[title]; ok {
			continue
		}
		values[title] = secret
		keys = append(keys, title)
	}
	sort.Strings(keys)
	return values, keys, nil
}

// emitMap prints export lines or JSON to a shell, and copies JSON in a
// terminal.
func (r Runner) emitMap(values map[string]string, keys []string) int {
	shell := r.forShell()
	if shell && allShellNames(keys) {
		for _, key := range keys {
			_, _ = fmt.Fprintf(r.Out, "export %s=%s\n", key, shellQuote(values[key]))
		}
		return 0
	}
	encoded, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return r.fail("Unable to encode the secret map.")
	}
	if shell {
		_, _ = r.Out.Write(append(encoded, '\n'))
		return 0
	}
	if err := r.Clip.Copy(string(encoded), r.Err); err != nil {
		return r.fail(err.Error())
	}
	_, _ = fmt.Fprintf(r.Out, "Copied %d secrets to the clipboard.\n", len(keys))
	return 0
}

func allShellNames(keys []string) bool {
	for _, key := range keys {
		if !shellIdentifier(key) {
			return false
		}
	}
	return len(keys) > 0
}

func shellIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		if !identByte(name[i], i == 0) {
			return false
		}
	}
	return true
}

func identByte(c byte, first bool) bool {
	letter := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
	if letter || c == '_' {
		return true
	}
	return !first && c >= '0' && c <= '9'
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
