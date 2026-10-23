// Package oauth keeps Gmail tokens in the google-auth authorized-user
// JSON format and runs the installed-app loopback flow.
package oauth

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/gmail/internal/config"
	"github.com/ryangerardwilson/c3po/gmail/internal/errs"
)

// ProfileURL is the Gmail profile used to learn the authorized address.
const ProfileURL = "https://gmail.googleapis.com/gmail/v1/users/me/profile"

// Scopes are the Gmail scopes the original CLI requested.
var Scopes = []string{
	"https://www.googleapis.com/auth/gmail.send",
	"https://www.googleapis.com/auth/gmail.readonly",
	"https://www.googleapis.com/auth/gmail.modify",
}

// Options carries the dependencies of token loading and authorization.
type Options struct {
	HTTP        *http.Client
	Paths       config.Paths
	ProfileURL  string
	Now         func() time.Time
	OpenBrowser func(url string) error
	Out         io.Writer
	TokenFile   func(email string) string
}

func (o Options) now() time.Time {
	if o.Now == nil {
		return time.Now()
	}
	return o.Now()
}

func (o Options) client() *http.Client {
	if o.HTTP != nil {
		return o.HTTP
	}
	return http.DefaultClient
}

func (o Options) tokenFile(email string) string {
	if o.TokenFile != nil {
		return o.TokenFile(email)
	}
	return o.Paths.TokenFile(email)
}

// Authorized is the result of a browser authorization.
type Authorized struct {
	Email string
	Token *Token
}

// Authorize runs the browser flow, reads the Gmail profile, and saves
// the token as tokens/<email>.json.
func (o Options) Authorize(clientSecretFile string) (Authorized, error) {
	if err := o.Paths.EnsureTokenDirs(); err != nil {
		return Authorized{}, err
	}
	cfg, err := readClientSecret(clientSecretFile)
	if err != nil {
		return Authorized{}, errs.API("OAuth authorization failed. Ensure the account is an allowed test user and the Gmail API is enabled. Details: %s", err.Error())
	}
	tok, err := o.runFlow(cfg, Scopes)
	if err != nil {
		return Authorized{}, errs.API("OAuth authorization failed. Ensure the account is an allowed test user and the Gmail API is enabled. Details: %s", err.Error())
	}
	email, err := o.profileEmail(tok.Access())
	if err != nil {
		return Authorized{}, err
	}
	if email == "" {
		return Authorized{}, errs.API("Gmail profile lookup returned no email address")
	}
	if err := tok.Write(o.tokenFile(email)); err != nil {
		return Authorized{}, err
	}
	return Authorized{Email: email, Token: tok}, nil
}

func (o Options) profileEmail(access string) (string, error) {
	u := o.ProfileURL
	if u == "" {
		u = ProfileURL
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := o.client().Do(req)
	if err != nil {
		return "", errs.API("Failed to fetch Gmail profile after OAuth: %s", err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return "", errs.API("Failed to fetch Gmail profile after OAuth: %s", resp.Status)
	}
	var info struct {
		Email string `json:"emailAddress"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return "", errs.API("Failed to fetch Gmail profile after OAuth: %s", err.Error())
	}
	return config.NormalizeEmail(info.Email), nil
}

// AccessToken returns a usable access token for email. A saved token is
// refreshed when it is expired. A missing token or a refused refresh
// starts a new browser authorization.
func (o Options) AccessToken(email, clientSecretFile string) (string, error) {
	if err := o.Paths.EnsureTokenDirs(); err != nil {
		return "", err
	}
	access, err := o.savedToken(o.tokenFile(email))
	if err != nil || access != "" {
		return access, err
	}
	if _, err := os.Stat(clientSecretFile); err != nil {
		return "", errs.API("Missing client secret file: %s", clientSecretFile)
	}
	authorized, err := o.Authorize(clientSecretFile)
	if err != nil {
		return "", err
	}
	if authorized.Email != config.NormalizeEmail(email) {
		return "", errs.API("Authorized account email `%s` does not match preset email `%s`. Re-run `c3po gmail auth` for this account.", authorized.Email, email)
	}
	return authorized.Token.Access(), nil
}

// savedToken returns "" when a new authorization is needed.
func (o Options) savedToken(path string) (string, error) {
	tok, err := ReadToken(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", errs.API("Failed to refresh Gmail token: %s", err.Error())
	}
	valid, err := tok.Valid(o.now())
	if err != nil {
		return "", errs.API("Failed to refresh Gmail token: %s", err.Error())
	}
	if valid && tok.HasScopes(Scopes) {
		return tok.Access(), nil
	}
	if tok.Refresh() == "" {
		return "", nil
	}
	var want []string
	if !tok.HasScopes(Scopes) {
		want = Scopes
	}
	err = tok.refresh(o.client(), o.now(), want)
	if err != nil && strings.Contains(err.Error(), "token refresh refused") {
		return "", nil
	}
	if err != nil {
		return "", errs.API("Failed to refresh Gmail token: %s", err.Error())
	}
	if err := tok.Write(path); err != nil {
		return "", err
	}
	return tok.Access(), nil
}
