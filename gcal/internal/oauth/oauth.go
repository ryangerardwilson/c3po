package oauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ryangerardwilson/c3po/gcal/internal/errs"
	"github.com/ryangerardwilson/c3po/gcal/internal/paths"
)

// UserinfoURL is Google's OAuth2 v2 userinfo endpoint.
const UserinfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

// CalendarScopes are requested by auth and every calendar command.
var CalendarScopes = []string{
	"https://www.googleapis.com/auth/calendar.events",
	"https://www.googleapis.com/auth/userinfo.email",
	"openid",
}

// TranscriptScopes add Meet transcript reads in Drive to CalendarScopes.
var TranscriptScopes = append(append([]string{}, CalendarScopes...), "https://www.googleapis.com/auth/drive.meet.readonly")

// Options carries the dependencies of token loading and authorization.
type Options struct {
	HTTP        *http.Client
	Dirs        paths.Dirs
	UserinfoURL string
	Now         func() time.Time
	OpenBrowser func(url string) error
	Out         io.Writer
}

func (o Options) now() time.Time {
	if o.Now == nil {
		return time.Now()
	}
	return o.Now()
}

// Authorized is the result of a browser authorization.
type Authorized struct {
	Email string
	Token *Token
}

// Authorize runs the browser flow, finds the account email, and saves
// the token as tokens/<email>.json.
func (o Options) Authorize(clientSecretFile string, scopes []string) (Authorized, error) {
	if err := o.Dirs.Ensure(); err != nil {
		return Authorized{}, err
	}
	cfg, err := readClientSecret(clientSecretFile)
	if err != nil {
		return Authorized{}, err
	}
	tok, err := o.runFlow(cfg, scopes)
	if err != nil {
		return Authorized{}, err
	}
	email, err := o.userEmail(tok.Access())
	if err != nil {
		return Authorized{}, err
	}
	if err := tok.Write(o.Dirs.TokenFile(email)); err != nil {
		return Authorized{}, err
	}
	return Authorized{Email: email, Token: tok}, nil
}

func (o Options) userEmail(access string) (string, error) {
	u := o.UserinfoURL
	if u == "" {
		u = UserinfoURL
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := o.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("userinfo request failed: %s", resp.Status)
	}
	var info struct {
		Email any `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", err
	}
	email := ""
	if info.Email != nil {
		email = paths.NormalizeEmail(fmt.Sprint(info.Email))
	}
	if email == "" {
		return "", errs.API("Authorized Google account email could not be determined.")
	}
	return email, nil
}

// AccessToken returns a usable access token for the account. A saved
// token is used while valid, refreshed when expired, and replaced by a
// new browser authorization when it is missing, lacks a scope, or the
// refresh is refused.
func (o Options) AccessToken(email, clientSecretFile string, scopes []string) (string, error) {
	if err := o.Dirs.Ensure(); err != nil {
		return "", err
	}
	access, err := o.savedToken(o.Dirs.TokenFile(email), scopes)
	if err != nil || access != "" {
		return access, err
	}
	if _, err := os.Stat(clientSecretFile); err != nil {
		return "", errs.API(fmt.Sprintf("Missing client secret file: %s", clientSecretFile))
	}
	authorized, err := o.Authorize(clientSecretFile, scopes)
	if err != nil {
		return "", err
	}
	if authorized.Email != email {
		return "", errs.API(fmt.Sprintf("Authorized account email `%s` does not match preset email `%s`. Re-run `c3po gcal auth` for this account.", authorized.Email, email))
	}
	return authorized.Token.Access(), nil
}

// savedToken returns "" with no error when a new authorization is needed.
// A token whose recorded scopes miss one of scopes is refreshed with
// those scopes, so a refresh token that already holds them still works.
func (o Options) savedToken(path string, scopes []string) (string, error) {
	tok, err := ReadToken(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	covered := tok.HasScopes(scopes)
	valid, err := tok.Valid(o.now())
	if err != nil || (valid && covered) {
		return tok.Access(), err
	}
	if tok.Refresh() == "" {
		return "", nil
	}
	var want []string
	if !covered {
		want = scopes
	}
	err = tok.refresh(o.HTTP, o.now(), want)
	if errors.Is(err, errRefresh) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return tok.Access(), tok.Write(path)
}
