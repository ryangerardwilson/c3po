package cli

import (
	"fmt"
	"strings"

	"github.com/ryangerardwilson/c3po/pass/internal/item"
	"github.com/ryangerardwilson/c3po/pass/internal/op"
	"github.com/ryangerardwilson/c3po/pass/internal/vault"
)

const (
	createUsage = "Use: c3po pass <vault> --create --title <title> --category api-credential|login [--username <user>]"
	editUsage   = "Use: c3po pass <vault> --edit <title> [--title <title>] [--username <user>]"
	bothUsage   = "Use the template or the flags, not both."
)

func (r Runner) createItem(name string, args []string) int {
	name, client, ok := r.begin(name)
	if !ok {
		return 1
	}
	positionals, flags, err := parseItemFlags(args)
	if err != nil {
		return r.fail(err.Error())
	}
	if len(positionals) != 0 {
		return r.fail(createUsage)
	}
	if path, ok := flags["template"]; ok {
		if flags["title"] != "" || flags["category"] != "" || flags["username"] != "" {
			return r.fail(bothUsage)
		}
		return r.createFromTemplate(client, name, path)
	}
	return r.createFromFlags(client, name, flags)
}

func (r Runner) createFromFlags(client op.Client, name string, flags map[string]string) int {
	category, problem := flagCategory(flags)
	if problem != "" {
		return r.fail(problem)
	}
	secret, found, err := r.readSecret()
	if err != nil {
		return r.fail("Unable to read the secret.")
	}
	if !found {
		return r.fail("Pass the secret on stdin.")
	}
	if !r.titleFree(client, name, flags["title"]) {
		return 1
	}
	body, err := item.New(flags["title"], category, flags["username"], secret)
	if err != nil {
		return r.fail("Unable to create the 1Password item.")
	}
	return r.create(client, name, body, category, flags["title"])
}

// flagCategory returns the op category of a flag create, or the message
// when the flags cannot create an item.
func flagCategory(flags map[string]string) (string, string) {
	if flags["title"] == "" || flags["category"] == "" {
		return "", createUsage
	}
	category, err := item.NormalizeCategory(flags["category"])
	if err != nil {
		return "", err.Error()
	}
	if category == "LOGIN" && flags["username"] == "" {
		return "", "A login needs --username."
	}
	return category, ""
}

func (r Runner) createFromTemplate(client op.Client, name, path string) int {
	body, ok := r.templateBytes(path)
	if !ok {
		return 1
	}
	title, category, err := item.TemplateMeta(body)
	if err != nil {
		return r.fail(err.Error())
	}
	if !r.titleFree(client, name, title) {
		return 1
	}
	return r.create(client, name, body, category, title)
}

// create sends body to op item create on stdin.
func (r Runner) create(client op.Client, name string, body []byte, category, title string) int {
	if err := client.Do(body, "item", "create", "--vault", name, "-"); err != nil {
		return r.fail("Unable to create the 1Password item.")
	}
	_, _ = fmt.Fprintf(r.Out, "Created %s %s in vault %s.\n", item.CategoryLabel(category), title, name)
	return 0
}

func (r Runner) editItem(name string, args []string) int {
	name, client, ok := r.begin(name)
	if !ok {
		return 1
	}
	positionals, flags, err := parseItemFlags(args)
	if err != nil {
		return r.fail(err.Error())
	}
	if len(positionals) != 1 || strings.TrimSpace(positionals[0]) == "" {
		return r.fail(editUsage)
	}
	current := strings.TrimSpace(positionals[0])
	found, ok := r.oneExactItem(client, name, current)
	if !ok {
		return 1
	}
	if path, ok := flags["template"]; ok {
		if flags["title"] != "" || flags["username"] != "" {
			return r.fail(bothUsage)
		}
		return r.editFromTemplate(client, name, found, path)
	}
	return r.editFields(client, name, found, current, flags)
}

func (r Runner) editFromTemplate(client op.Client, name string, found op.Item, path string) int {
	body, ok := r.templateBytes(path)
	if !ok {
		return 1
	}
	if err := item.CheckObject(body); err != nil {
		return r.fail(err.Error())
	}
	return r.edit(client, name, found.ID, body, found.Title)
}

func (r Runner) editFields(client op.Client, name string, found op.Item, current string, flags map[string]string) int {
	secret, hasSecret, err := r.readSecret()
	if err != nil {
		return r.fail("Unable to read the secret.")
	}
	if problem := editProblem(found.Category, flags, hasSecret); problem != "" {
		return r.fail(problem)
	}
	doc, err := client.Get(name, found.ID)
	if err != nil {
		return r.fail(err.Error())
	}
	title := or(found.Title, current)
	if flags["title"] != "" {
		title = flags["title"]
		doc["title"] = title
	}
	if flags["username"] != "" {
		item.SetUsername(doc, flags["username"])
	}
	if hasSecret {
		item.SetSecret(doc, found.Category, secret)
	}
	body, err := item.Marshal(doc)
	if err != nil {
		return r.fail("Unable to update the 1Password item.")
	}
	return r.edit(client, name, found.ID, body, title)
}

// editProblem is the message when an edit has nothing to do or cannot
// apply, and empty when it can.
func editProblem(category string, flags map[string]string, hasSecret bool) string {
	if !hasSecret && flags["title"] == "" && flags["username"] == "" {
		return "Nothing to change."
	}
	if hasSecret && !item.SecretCategory(category) {
		return "This category has no single secret field. Use --template."
	}
	if flags["username"] != "" && !item.UsernameCategory(category) {
		return "--username applies to an API credential or a login. Use --template for another category."
	}
	return ""
}

// edit sends body to op item edit on stdin.
func (r Runner) edit(client op.Client, name, id string, body []byte, title string) int {
	if err := client.Do(body, "item", "edit", id, "--vault", name); err != nil {
		return r.fail("Unable to update the 1Password item.")
	}
	_, _ = fmt.Fprintf(r.Out, "Updated %s in vault %s.\n", title, name)
	return 0
}

func (r Runner) deleteItem(name string, args []string) int {
	name, client, ok := r.begin(name)
	if !ok {
		return 1
	}
	if !oneTitleArg(args) {
		return r.fail("Use: c3po pass <vault> --delete <title>")
	}
	found, ok := r.oneExactItem(client, name, strings.TrimSpace(args[0]))
	if !ok {
		return 1
	}
	if err := client.Do(nil, "item", "delete", found.ID, "--vault", name); err != nil {
		return r.fail("Unable to delete the 1Password item.")
	}
	_, _ = fmt.Fprintf(r.Out, "Deleted %s from vault %s.\n", found.Title, name)
	return 0
}

func (r Runner) showItem(name string, args []string) int {
	name, client, ok := r.begin(name)
	if !ok {
		return 1
	}
	if !oneTitleArg(args) {
		return r.fail("Use: c3po pass <vault> --show <title>")
	}
	found, ok := r.oneExactItem(client, name, strings.TrimSpace(args[0]))
	if !ok {
		return 1
	}
	doc, err := client.Get(name, found.ID)
	if err != nil {
		return r.fail(err.Error())
	}
	item.WriteShow(r.Out, doc, found.Title, found.ID, found.Category)
	return 0
}

func oneTitleArg(args []string) bool {
	return len(args) == 1 && strings.TrimSpace(args[0]) != "" && !strings.HasPrefix(args[0], "--")
}

// titleFree fails when vault already has an item with that title.
func (r Runner) titleFree(client op.Client, name, title string) bool {
	items, err := client.List(name)
	if err != nil {
		r.fail(err.Error())
		return false
	}
	if len(vault.Exact(items, title)) > 0 {
		r.fail(fmt.Sprintf("An item titled %s already exists in vault %s.", title, name))
		return false
	}
	return true
}

// oneExactItem returns the one item titled title, without case.
func (r Runner) oneExactItem(client op.Client, name, title string) (op.Item, bool) {
	items, err := client.List(name)
	if err != nil {
		r.fail(err.Error())
		return op.Item{}, false
	}
	hits := vault.Exact(items, title)
	switch len(hits) {
	case 1:
		return hits[0], true
	case 0:
		r.fail(fmt.Sprintf("No item titled %s in vault %s.", title, name))
	default:
		r.fail(fmt.Sprintf("More than one item titled %s in vault %s.", title, name))
	}
	return op.Item{}, false
}

func or(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
