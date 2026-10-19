// Package query turns list time windows into Gmail search terms.
package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
)

const shape = `2w | 14d | 3m | 1y | 2025-01 | "jan 2025" | 2025-01-10 | 2025-01-10..2025-01-20`

var (
	relativeRe = regexp.MustCompile(`^(?P<amount>\d+)(?P<unit>[dwmy])$`)
	isoMonthRe = regexp.MustCompile(`^(?P<year>\d{4})-(?P<month>\d{2})$`)
	isoDateRe  = regexp.MustCompile(`^(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})$`)
	namedRe    = regexp.MustCompile(`^(?P<month>[A-Za-z]+)[ -]+(?P<year>\d{4})$`)
)

var months = map[string]time.Month{
	"jan": time.January, "january": time.January,
	"feb": time.February, "february": time.February,
	"mar": time.March, "march": time.March,
	"apr": time.April, "april": time.April,
	"may": time.May,
	"jun": time.June, "june": time.June,
	"jul": time.July, "july": time.July,
	"aug": time.August, "august": time.August,
	"sep": time.September, "sept": time.September, "september": time.September,
	"oct": time.October, "october": time.October,
	"nov": time.November, "november": time.November,
	"dec": time.December, "december": time.December,
}

// TimeLimit converts one since-window into a Gmail search fragment.
func TimeLimit(value string) (string, error) {
	expr := strings.TrimSpace(value)
	if expr == "" {
		return "", errs.Usage("ls -tl requires: <time_limit>")
	}
	if strings.Contains(expr, "..") {
		return dateRange(expr)
	}
	if term, ok, err := relative(expr); ok || err != nil {
		return term, err
	}
	if start, end, ok, err := month(expr); ok || err != nil {
		if err != nil {
			return "", err
		}
		return inclusive(start, end)
	}
	if start, end, ok, err := namedMonth(expr); ok || err != nil {
		if err != nil {
			return "", err
		}
		return inclusive(start, end)
	}
	if day, ok, err := isoDate(expr); ok || err != nil {
		if err != nil {
			return "", err
		}
		return inclusive(day, day)
	}
	return "", errs.Usage("ls -tl supports: %s", shape)
}

func dateRange(expr string) (string, error) {
	startRaw, endRaw, ok := strings.Cut(expr, "..")
	if !ok || strings.TrimSpace(startRaw) == "" || strings.TrimSpace(endRaw) == "" {
		return "", errs.Usage("ls -tl supports: %s", shape)
	}
	start, sok, serr := isoDate(strings.TrimSpace(startRaw))
	end, eok, eerr := isoDate(strings.TrimSpace(endRaw))
	if serr != nil {
		return "", serr
	}
	if eerr != nil {
		return "", eerr
	}
	if !sok || !eok {
		return "", errs.Usage("ls -tl date ranges must use: YYYY-MM-DD..YYYY-MM-DD")
	}
	return inclusive(start, end)
}

func relative(expr string) (string, bool, error) {
	m := relativeRe.FindStringSubmatch(strings.ToLower(expr))
	if m == nil {
		return "", false, nil
	}
	amount, _ := strconv.Atoi(m[1])
	if amount <= 0 {
		return "", true, errs.Usage("ls -tl duration must be > 0")
	}
	unit := m[2]
	if unit == "w" {
		return fmt.Sprintf("newer_than:%dd", amount*7), true, nil
	}
	return fmt.Sprintf("newer_than:%d%s", amount, unit), true, nil
}

func month(expr string) (time.Time, time.Time, bool, error) {
	m := isoMonthRe.FindStringSubmatch(expr)
	if m == nil {
		return time.Time{}, time.Time{}, false, nil
	}
	year, _ := strconv.Atoi(m[1])
	mon, _ := strconv.Atoi(m[2])
	start, end, err := bounds(year, time.Month(mon))
	if err != nil {
		return time.Time{}, time.Time{}, true, errs.Usage("Invalid ls -tl month '%s'", expr)
	}
	return start, end, true, nil
}

func namedMonth(expr string) (time.Time, time.Time, bool, error) {
	m := namedRe.FindStringSubmatch(strings.TrimSpace(expr))
	if m == nil {
		return time.Time{}, time.Time{}, false, nil
	}
	mon, ok := months[strings.ToLower(m[1])]
	if !ok {
		return time.Time{}, time.Time{}, true, errs.Usage("Invalid ls -tl month '%s'", expr)
	}
	year, _ := strconv.Atoi(m[2])
	start, end, err := bounds(year, mon)
	if err != nil {
		return time.Time{}, time.Time{}, true, errs.Usage("Invalid ls -tl month '%s'", expr)
	}
	return start, end, true, nil
}

func isoDate(expr string) (time.Time, bool, error) {
	m := isoDateRe.FindStringSubmatch(expr)
	if m == nil {
		return time.Time{}, false, nil
	}
	year, _ := strconv.Atoi(m[1])
	mon, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])
	t := time.Date(year, time.Month(mon), day, 0, 0, 0, 0, time.UTC)
	if t.Year() != year || int(t.Month()) != mon || t.Day() != day {
		return time.Time{}, true, errs.Usage("Invalid ls -tl date '%s'", expr)
	}
	return t, true, nil
}

func bounds(year int, month time.Month) (time.Time, time.Time, error) {
	if month < 1 || month > 12 {
		return time.Time{}, time.Time{}, fmt.Errorf("bad month")
	}
	start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0).AddDate(0, 0, -1)
	return start, end, nil
}

func inclusive(start, end time.Time) (string, error) {
	if end.Before(start) {
		return "", errs.Usage("ls -tl range end must be on or after start")
	}
	return fmt.Sprintf("after:%s before:%s", gmailDate(start.AddDate(0, 0, -1)), gmailDate(end.AddDate(0, 0, 1))), nil
}

func gmailDate(t time.Time) string {
	return t.Format("2006/01/02")
}
