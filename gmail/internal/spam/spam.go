// Package spam selects spam senders and builds the cleanup queries.
package spam

import "strings"

// AlwaysDomains are trashed on every clean even when the config omits them.
var AlwaysDomains = []string{"@clickup.com", "@mail.clickup.com"}

// Count is one sender and its unread count.
type Count struct {
	Sender string
	Unread int
}

// Candidates returns senders over threshold, excluding gmail, the preset
// domain, existing spam, and spam_excludes. Highest count first.
func Candidates(counts map[string]int, existing, excludes []string, presetEmail string, threshold int) []Count {
	spamSet := setOf(existing)
	excludeSet := setOf(excludes)
	presetDomain := domainOf(presetEmail)
	var selected []Count
	for sender, n := range counts {
		if n <= threshold || inTargets(sender, spamSet) || inTargets(sender, excludeSet) || excludedSender(sender, presetDomain) {
			continue
		}
		selected = append(selected, Count{Sender: sender, Unread: n})
	}
	sortCounts(selected)
	return selected
}

func excludedSender(sender, presetDomain string) bool {
	lower := strings.ToLower(sender)
	if strings.HasSuffix(lower, "@gmail.com") {
		return true
	}
	return presetDomain != "" && strings.HasSuffix(lower, "@"+presetDomain)
}

// WithAlways appends the always-spam domains that are not already listed.
func WithAlways(senders []string) []string {
	seen := map[string]bool{}
	for _, item := range senders {
		item = strings.ToLower(strings.TrimSpace(item))
		if item != "" {
			seen[item] = true
		}
	}
	out := append([]string(nil), senders...)
	for _, domain := range AlwaysDomains {
		if !seen[domain] {
			out = append(out, domain)
			seen[domain] = true
		}
	}
	return out
}

// Groups splits exact senders into chunks and gives each domain its own group.
func Groups(senders []string, size int) [][]string {
	var domains, exact []string
	for _, item := range senders {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "@") {
			domains = append(domains, item)
			continue
		}
		exact = append(exact, item)
	}
	var groups [][]string
	if size <= 0 {
		if len(exact) > 0 {
			groups = append(groups, exact)
		}
	} else {
		for i := 0; i < len(exact); i += size {
			end := i + size
			if end > len(exact) {
				end = len(exact)
			}
			groups = append(groups, exact[i:end])
		}
	}
	for _, item := range domains {
		groups = append(groups, []string{item})
	}
	return groups
}

// Query is the Gmail search for one sender group.
func Query(senders []string) string {
	if len(senders) == 1 {
		return fromTerm(senders[0])
	}
	terms := make([]string, len(senders))
	for i, sender := range senders {
		terms[i] = fromTerm(sender)
	}
	return "(" + strings.Join(terms, " OR ") + ")"
}

func fromTerm(sender string) string {
	sender = strings.TrimSpace(sender)
	if strings.HasPrefix(sender, "@") {
		return "from:" + sender[1:]
	}
	return "from:" + sender
}

// CleanupSenders drops senders that match an exclude.
func CleanupSenders(senders, excludes []string) []string {
	blocked := setOf(excludes)
	var out []string
	for _, sender := range WithAlways(senders) {
		if !inTargets(sender, blocked) {
			out = append(out, sender)
		}
	}
	return out
}

func setOf(items []string) map[string]bool {
	out := map[string]bool{}
	for _, item := range items {
		item = strings.ToLower(strings.TrimSpace(item))
		if item != "" {
			out[item] = true
		}
	}
	return out
}

func inTargets(sender string, targets map[string]bool) bool {
	for target := range targets {
		if matches(target, sender) {
			return true
		}
	}
	return false
}

func matches(target, sender string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	sender = strings.ToLower(strings.TrimSpace(sender))
	if target == "" || sender == "" {
		return false
	}
	if !strings.HasPrefix(target, "@") {
		return sender == target
	}
	domain := target[1:]
	if domain == "" {
		return false
	}
	if !strings.Contains(sender, "@") {
		return sender == domain || strings.HasSuffix(sender, "."+domain)
	}
	senderDomain := sender[strings.LastIndex(sender, "@")+1:]
	return senderDomain == domain || strings.HasSuffix(senderDomain, "."+domain)
}

func domainOf(email string) string {
	if !strings.Contains(email, "@") {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(strings.SplitN(email, "@", 2)[1]))
}

func sortCounts(items []Count) {
	for i := 1; i < len(items); i++ {
		j := i
		for j > 0 && lessCount(items[j], items[j-1]) {
			items[j], items[j-1] = items[j-1], items[j]
			j--
		}
	}
}

func lessCount(a, b Count) bool {
	if a.Unread != b.Unread {
		return a.Unread > b.Unread
	}
	return a.Sender < b.Sender
}
