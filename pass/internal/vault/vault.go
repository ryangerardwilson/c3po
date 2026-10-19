// Package vault matches listed 1Password items by title and username.
package vault

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ryangerardwilson/c3po/pass/internal/op"
)

// Match is one search hit.
type Match struct {
	ID       string
	VaultID  string
	Title    string
	Username string
	Category string
}

// Format is the numbered line printed for a hit.
func (m Match) Format(number int) string {
	if m.Username == "" {
		return fmt.Sprintf("%d) %s", number, m.Title)
	}
	return fmt.Sprintf("%d) %s | %s", number, m.Title, m.Username)
}

// DisplayTitle is the title, or (untitled).
func DisplayTitle(item op.Item) string {
	if item.Title == "" {
		return "(untitled)"
	}
	return item.Title
}

// Usernames returns the usernames of a Login item, and nothing for
// another category.
func Usernames(item op.Item) []string {
	if strings.ToLower(item.Category) != "login" {
		return nil
	}
	var values []string
	if item.AdditionalInformation != "" {
		values = append(values, item.AdditionalInformation)
	}
	for _, field := range item.Fields {
		label := field.Label
		if label == "" {
			label = field.Name
		}
		if strings.ToLower(label) == "username" && field.Value != "" {
			values = append(values, field.Value)
		}
	}
	return values
}

// Lines are the -l lines: title, then usernames for a Login, sorted
// without case.
func Lines(items []op.Item) []string {
	lines := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		if seen[item.ID] {
			continue
		}
		if item.ID != "" {
			seen[item.ID] = true
		}
		lines = append(lines, line(item))
	}
	sort.Slice(lines, func(i, j int) bool {
		return strings.ToLower(lines[i]) < strings.ToLower(lines[j])
	})
	return lines
}

func line(item op.Item) string {
	usernames := Usernames(item)
	if len(usernames) == 0 {
		return DisplayTitle(item)
	}
	return DisplayTitle(item) + " | " + strings.Join(usernames, ", ")
}

// Find returns items whose title or Login username contains search,
// without case.
func Find(items []op.Item, search string) []Match {
	needle := strings.ToLower(search)
	seen := map[string]bool{}
	var matches []Match
	for _, item := range items {
		if item.ID == "" || seen[item.ID] {
			continue
		}
		usernames := Usernames(item)
		title := DisplayTitle(item)
		if !strings.Contains(strings.ToLower(title), needle) && !containsFold(usernames, needle) {
			continue
		}
		seen[item.ID] = true
		matches = append(matches, Match{
			ID:       item.ID,
			VaultID:  item.Vault.ID,
			Title:    title,
			Username: strings.Join(usernames, ", "),
			Category: item.Category,
		})
	}
	return matches
}

func containsFold(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), needle) {
			return true
		}
	}
	return false
}

// IsGlob reports whether a name is a glob.
func IsGlob(selector string) bool {
	return strings.ContainsAny(selector, "*?[")
}

// Select returns the items named by selectors. A plain name must match
// one title exactly, without case. A glob must match at least one.
func Select(items []op.Item, selectors []string) ([]op.Item, error) {
	var chosen []op.Item
	seen := map[string]bool{}
	for _, raw := range selectors {
		hits, err := hitsFor(items, strings.TrimSpace(raw))
		if err != nil {
			return nil, err
		}
		for _, item := range hits {
			if seen[item.ID] {
				continue
			}
			if item.ID != "" {
				seen[item.ID] = true
			}
			chosen = append(chosen, item)
		}
	}
	return chosen, nil
}

func hitsFor(items []op.Item, selector string) ([]op.Item, error) {
	if selector == "" {
		return nil, errors.New("Item name must not be empty.")
	}
	if IsGlob(selector) {
		hits := matchTitles(items, selector, true)
		if len(hits) == 0 {
			return nil, fmt.Errorf("No matches for %s.", selector)
		}
		return hits, nil
	}
	hits := matchTitles(items, selector, false)
	switch len(hits) {
	case 0:
		return nil, fmt.Errorf("No item named %s.", selector)
	case 1:
		return hits, nil
	default:
		return nil, fmt.Errorf("More than one item named %s.", selector)
	}
}

func matchTitles(items []op.Item, selector string, glob bool) []op.Item {
	var hits []op.Item
	pattern := strings.ToLower(selector)
	for _, item := range items {
		if titleMatches(strings.ToLower(DisplayTitle(item)), pattern, glob) {
			hits = append(hits, item)
		}
	}
	return hits
}

func titleMatches(title, pattern string, glob bool) bool {
	if !glob {
		return title == pattern
	}
	ok, err := filepath.Match(pattern, title)
	return err == nil && ok
}

// Exact returns the items whose trimmed title equals title, without case.
func Exact(items []op.Item, title string) []op.Item {
	title = strings.TrimSpace(title)
	var hits []op.Item
	seen := map[string]bool{}
	for _, item := range items {
		if item.ID == "" || seen[item.ID] {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(DisplayTitle(item)), title) {
			seen[item.ID] = true
			hits = append(hits, item)
		}
	}
	return hits
}
