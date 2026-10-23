// Package oauth keeps Google OAuth tokens in the google-auth
// "authorized user" JSON format the Python gcal wrote, refreshes them
// through their token_uri, and runs the installed-app loopback flow.
package oauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/gmail/internal/pyjson"
	"github.com/ryangerardwilson/c3po/internal/sqlxdb"
)

// GoogleTokenURI is used when a token file has no token_uri.
const GoogleTokenURI = "https://oauth2.googleapis.com/token"

// refreshThreshold matches google-auth: a token this close to expiry is
// treated as expired.
const refreshThreshold = 3*time.Minute + 45*time.Second

// fieldOrder is the key order google-auth Credentials.to_json writes.
var fieldOrder = []string{
	"token", "refresh_token", "token_uri", "client_id", "client_secret",
	"scopes", "rapt_token", "universe_domain", "account", "expiry",
}

// Token is one authorized-user token file. Unknown keys are kept.
type Token struct {
	fields map[string]any
}

// NewToken builds a token like google-auth does after an OAuth flow.
func NewToken(access, refresh, tokenURI, clientID, clientSecret string, scopes []string, expiry time.Time) *Token {
	t := &Token{fields: map[string]any{
		"token":           access,
		"token_uri":       tokenURI,
		"client_id":       clientID,
		"client_secret":   clientSecret,
		"scopes":          toAny(scopes),
		"universe_domain": "googleapis.com",
		"account":         "",
	}}
	if refresh != "" {
		t.fields["refresh_token"] = refresh
	}
	t.SetExpiry(expiry)
	return t
}

func toAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// ReadToken loads a token file. It fails like google-auth when
// refresh_token, client_id, or client_secret is missing.
func ReadToken(path string) (*Token, error) {
	raw, err := sqlxdb.Read("gmail", "token", sqlxdb.Key(path), path)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&fields); err != nil {
		return nil, fmt.Errorf("invalid token file %s: %w", path, err)
	}
	var missing []string
	for _, k := range []string{"client_id", "client_secret", "refresh_token"} {
		if _, ok := fields[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("Authorized user info was not in the expected format, missing fields %s.", strings.Join(missing, ", "))
	}
	return &Token{fields: fields}, nil
}

func (t *Token) str(key string) string {
	s, _ := t.fields[key].(string)
	return s
}

// Access is the access token.
func (t *Token) Access() string { return t.str("token") }

// Refresh is the refresh token.
func (t *Token) Refresh() string { return t.str("refresh_token") }

// URI is the token endpoint, Google's when the file has none.
func (t *Token) URI() string {
	if u := t.str("token_uri"); u != "" {
		return u
	}
	return GoogleTokenURI
}

// Scopes lists the recorded scopes and whether the file had any.
func (t *Token) Scopes() ([]string, bool) {
	switch v := t.fields["scopes"].(type) {
	case string:
		return strings.Fields(v), true
	case []any:
		out := make([]string, 0, len(v))
		for _, s := range v {
			if str, ok := s.(string); ok {
				out = append(out, str)
			}
		}
		return out, true
	default:
		return nil, false
	}
}

// HasScopes reports whether every wanted scope is recorded. A file with
// no scopes key is trusted, since google-auth never checked it either.
func (t *Token) HasScopes(want []string) bool {
	have, ok := t.Scopes()
	if !ok {
		return true
	}
	set := map[string]bool{}
	for _, s := range have {
		set[s] = true
	}
	for _, s := range want {
		if !set[s] {
			return false
		}
	}
	return true
}

// Expiry parses the expiry like google-auth: naive UTC, fraction and Z
// dropped. ok is false when the file has no expiry.
func (t *Token) Expiry() (time.Time, bool, error) {
	raw := t.str("expiry")
	if raw == "" {
		return time.Time{}, false, nil
	}
	base, _, _ := strings.Cut(strings.TrimRight(raw, "Z"), ".")
	exp, err := time.Parse("2006-01-02T15:04:05", base)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("invalid token expiry %q", raw)
	}
	return exp, true, nil
}

// SetExpiry records expiry, or drops it for the zero time.
func (t *Token) SetExpiry(exp time.Time) {
	if exp.IsZero() {
		delete(t.fields, "expiry")
		return
	}
	exp = exp.UTC()
	s := exp.Format("2006-01-02T15:04:05")
	if us := exp.Nanosecond() / 1000; us != 0 {
		s += fmt.Sprintf(".%06d", us)
	}
	t.fields["expiry"] = s + "Z"
}

// Valid reports an access token that is not expired at now. A token
// with no expiry is treated as expired, like google-auth.
func (t *Token) Valid(now time.Time) (bool, error) {
	if t.Access() == "" {
		return false, nil
	}
	exp, ok, err := t.Expiry()
	if err != nil || !ok {
		return false, err
	}
	return now.UTC().Before(exp.Add(-refreshThreshold)), nil
}

// Write saves the token in google-auth's to_json layout.
func (t *Token) Write(path string) error {
	obj := pyjson.NewObject()
	seen := map[string]bool{}
	for _, k := range fieldOrder {
		seen[k] = true
		if v, ok := t.fields[k]; ok && v != nil {
			obj.Set(k, plain(v))
		}
	}
	var extra []string
	for k := range t.fields {
		if !seen[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range extra {
		obj.Set(k, plain(t.fields[k]))
	}
	body := []byte(pyjson.Dumps(obj, -1, false))
	return sqlxdb.Write("gmail", "token", sqlxdb.Key(path), path, body, 0o600)
}

// plain turns decoded JSON numbers into values pyfmt can print.
func plain(v any) any {
	switch x := v.(type) {
	case json.Number:
		if f, err := x.Float64(); err == nil {
			return f
		}
		return x.String()
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = plain(e)
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for k, e := range x {
			out[k] = plain(e)
		}
		return out
	default:
		return v
	}
}
