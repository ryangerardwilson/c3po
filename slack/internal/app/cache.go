package app

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type cachePaths struct {
	DBFile string
}

func safePresetSlug(preset string) string {
	re := regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	slug := re.ReplaceAllString(preset, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "default"
	}
	return slug
}

func accountInt(account map[string]any, key string, fallback int) int {
	return intValue(account[key], fallback)
}

func eventCachePaths(account map[string]any, preset string) cachePaths {
	slug := safePresetSlug(preset)
	base := stateBaseDir()
	dbPath := firstNonEmpty(str(account["events_cache_db"]), str(account["event_cache_db"]), filepath.Join(base, "events-"+slug+".db"))
	return cachePaths{DBFile: expandPath(dbPath)}
}

func eventCacheDBPath(account map[string]any, preset string) string {
	return eventCachePaths(account, preset).DBFile
}

func (rt *Runtime) eventCacheLoadConversationRows(cachePath, selfUserID string, limit int) ([]ConversationRow, error) {
	entries, err := rt.Cache.LoadEntries(cachePath, selfUserID, maxInt(500, limit*5), "")
	if err != nil || len(entries) == 0 {
		return nil, err
	}
	rows := conversationRowsFromEntries(entries)
	var filtered []ConversationRow
	history := rt.Cache.HistoryLoaded(cachePath)
	for _, row := range rows {
		if row.Surface == "dm" || row.Surface == "group_dm" {
			row.HistoryLoaded = history[row.ChannelID]
			filtered = append(filtered, row)
		}
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func (rt *Runtime) eventCacheSearchEntries(cachePath string, contacts Contacts, limit int, filterMode, selfUserID, label, senderFilter, containsFilter, timeLimit, channelFilter string) ([]MessageEntry, error) {
	entries, err := rt.Cache.LoadEntries(cachePath, selfUserID, maxInt(200, limit*20), "")
	if err != nil || len(entries) == 0 {
		return entries, err
	}
	cutoff, hasCutoff := startTS(timeLimit, rt.Now())
	if strings.TrimSpace(timeLimit) != "" && !hasCutoff {
		return nil, UsageError{Message: "invalid since window " + strconv.Quote(timeLimit)}
	}
	var selected []MessageEntry
	sortEntriesLatest(entries)
	for _, entry := range entries {
		if !cacheLabelMatches(entry, contacts, label) {
			continue
		}
		if !channelFilterMatches(entry, channelFilter) {
			continue
		}
		if !entryPassesFilters(entry, contacts, filterMode, senderFilter, containsFilter, cutoff, hasCutoff) {
			continue
		}
		selected = append(selected, entry)
		if len(selected) >= limit {
			break
		}
	}
	// Keep newest-first for agent-friendly "latest messages" workflows.
	return selected, nil
}

func channelFilterMatches(entry MessageEntry, channelFilter string) bool {
	channelFilter = strings.TrimSpace(channelFilter)
	if channelFilter == "" {
		return true
	}
	if conversationIDRE.MatchString(channelFilter) {
		return entry.ChannelID == channelFilter
	}
	name := channelNameQuery(channelFilter)
	if name == "" {
		name = strings.TrimPrefix(channelFilter, "#")
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	haystack := strings.ToLower(strings.Join([]string{entry.ChannelID, entry.Conversation, entry.Surface}, " "))
	return strings.Contains(haystack, name) || strings.EqualFold(entry.Conversation, channelFilter) || strings.EqualFold(entry.Conversation, "#"+name)
}

func cacheLabelMatches(entry MessageEntry, contacts Contacts, label string) bool {
	if label == "" {
		return true
	}
	target, ok := contacts[label]
	if !ok {
		return false
	}
	haystack := strings.ToLower(strings.Join([]string{
		label, target, entry.Conversation, entry.Email, entry.UserID,
		str(entry.Sender["id"]), str(entry.Sender["name"]), str(entry.Sender["email"]), str(entry.Sender["label"]),
	}, " "))
	target = strings.ToLower(target)
	return target != "" && strings.Contains(haystack, target) || strings.Contains(haystack, strings.ToLower(label))
}

func senderFilterTerms(contacts Contacts, senderFilter string) []string {
	senderFilter = strings.TrimSpace(senderFilter)
	if senderFilter == "" {
		return nil
	}
	terms := []string{strings.ToLower(senderFilter)}
	for _, key := range []string{senderFilter, strings.ToLower(senderFilter)} {
		target, ok := contacts[key]
		if !ok || target == "" {
			continue
		}
		terms = append(terms, strings.ToLower(target))
		if local := strings.Split(target, "@")[0]; local != "" {
			terms = append(terms, strings.ToLower(local))
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, term := range terms {
		if term == "" || seen[term] {
			continue
		}
		seen[term] = true
		out = append(out, term)
	}
	return out
}

func resolveSenderSearchTerm(contacts Contacts, senderFilter string) string {
	terms := senderFilterTerms(contacts, senderFilter)
	if len(terms) == 0 {
		return senderFilter
	}
	for _, term := range terms {
		if strings.Contains(term, "@") {
			return term
		}
	}
	return terms[0]
}

func entryPassesFilters(entry MessageEntry, contacts Contacts, filterMode, senderFilter, containsFilter string, cutoff float64, hasCutoff bool) bool {
	if filterMode == "unread" && !entry.Unread {
		return false
	}
	if filterMode == "read" && entry.Unread {
		return false
	}
	if senderFilter != "" {
		haystack := strings.ToLower(strings.Join([]string{
			str(entry.Sender["name"]), str(entry.Sender["email"]), str(entry.Sender["label"]), str(entry.Sender["id"]),
			entry.Conversation, entry.Email,
		}, " "))
		matched := false
		for _, term := range senderFilterTerms(contacts, senderFilter) {
			if strings.Contains(haystack, term) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if containsFilter != "" && !strings.Contains(strings.ToLower(messageText(entry.Message)), strings.ToLower(containsFilter)) {
		return false
	}
	if hasCutoff && entry.SortTS < cutoff {
		return false
	}
	return true
}

// startTS is the start of a since window, relative to now.
func startTS(value string, now time.Time) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if match := relativeTimeRE.FindStringSubmatch(value); len(match) == 3 {
		amount, err := strconv.Atoi(match[1])
		if err != nil || amount <= 0 {
			return 0, false
		}
		switch strings.ToLower(match[2]) {
		case "h":
			return float64(now.Add(-time.Duration(amount) * time.Hour).Unix()), true
		case "d":
			return float64(now.AddDate(0, 0, -amount).Unix()), true
		case "w":
			return float64(now.AddDate(0, 0, -amount*7).Unix()), true
		case "m":
			return float64(now.AddDate(0, -amount, 0).Unix()), true
		case "y":
			return float64(now.AddDate(-amount, 0, 0).Unix()), true
		}
	}
	if strings.Contains(value, "..") {
		value = strings.SplitN(value, "..", 2)[0]
	}
	layouts := []string{"2006-01-02", "2006-01"}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return float64(parsed.Unix()), true
		}
	}
	if match := namedMonthRE.FindStringSubmatch(value); len(match) == 3 {
		month := monthNumber(match[1])
		year, _ := strconv.Atoi(match[2])
		if month > 0 && year > 0 {
			return float64(time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local).Unix()), true
		}
	}
	_ = isoDateRE
	_ = isoMonthRE
	return 0, false
}

func formatCutoffUTC(cutoff float64) string {
	return time.Unix(int64(cutoff), 0).UTC().Format(time.RFC3339)
}

func monthNumber(value string) int {
	months := map[string]int{"jan": 1, "january": 1, "feb": 2, "february": 2, "mar": 3, "march": 3, "apr": 4, "april": 4, "may": 5, "jun": 6, "june": 6, "jul": 7, "july": 7, "aug": 8, "august": 8, "sep": 9, "sept": 9, "september": 9, "oct": 10, "october": 10, "nov": 11, "november": 11, "dec": 12, "december": 12}
	return months[strings.ToLower(value)]
}
