package config

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/gcal/internal/errs"
)

var offsetPattern = regexp.MustCompile(`^([+-])(\d{2}):?(\d{2})$`)

// NormalizeTimezone trims value and writes a UTC offset like +0530 as
// +05:30. IANA names are returned trimmed.
func NormalizeTimezone(value string) (string, error) {
	stripped := strings.TrimSpace(value)
	m := offsetPattern.FindStringSubmatch(stripped)
	if m == nil {
		return stripped, nil
	}
	hours, _ := strconv.Atoi(m[2])
	minutes, _ := strconv.Atoi(m[3])
	if hours > 23 || minutes > 59 {
		return "", errors.New("offset out of range")
	}
	return m[1] + m[2] + ":" + m[3], nil
}

// Location resolves an IANA name or a UTC offset to a time.Location.
func Location(value string) (*time.Location, error) {
	normalized, err := NormalizeTimezone(value)
	if err != nil {
		return nil, err
	}
	if m := offsetPattern.FindStringSubmatch(normalized); m != nil {
		hours, _ := strconv.Atoi(m[2])
		minutes, _ := strconv.Atoi(m[3])
		secs := hours*3600 + minutes*60
		if m[1] == "-" {
			secs = -secs
		}
		return time.FixedZone(normalized, secs), nil
	}
	if normalized == "" || normalized == "Local" {
		return nil, fmt.Errorf("unknown time zone %q", normalized)
	}
	return time.LoadLocation(normalized)
}

// ValidateTimezone checks a defaults.timezone value read from path. A
// missing value is UTC.
func ValidateTimezone(value any, path string) (string, error) {
	if value == nil {
		return "UTC", nil
	}
	s, ok := value.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "", errs.Config(fmt.Sprintf("Invalid config at %s: defaults.timezone must be a non-empty IANA timezone or UTC offset", path))
	}
	tz, err := NormalizeTimezone(s)
	if err == nil {
		_, err = Location(tz)
	}
	if err != nil {
		return "", errs.Config(fmt.Sprintf("Invalid config at %s: defaults.timezone must be a valid IANA timezone like 'Asia/Kolkata' or UTC offset like '+05:30'", path))
	}
	return tz, nil
}
