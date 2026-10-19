package oauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// errRefresh is a token endpoint refusal; the caller falls back to a
// new authorization, like google-auth's RefreshError.
var errRefresh = errors.New("token refresh refused")

type tokenResponse struct {
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	ExpiresIn    json.Number     `json:"expires_in"`
	Scope        string          `json:"scope"`
	Error        string          `json:"error"`
	ErrorDesc    string          `json:"error_description"`
}

// expiry is now plus expires_in, or zero when the response has none.
func (r tokenResponse) expiry(now time.Time) time.Time {
	secs, err := r.ExpiresIn.Float64()
	if r.ExpiresIn == "" || err != nil {
		return time.Time{}
	}
	return now.UTC().Add(time.Duration(secs * float64(time.Second)))
}

// postToken sends one form to a token endpoint. A non-2xx answer or an
// answer with no access_token wraps errRefresh.
func postToken(client *http.Client, uri string, form url.Values) (tokenResponse, error) {
	resp, err := client.PostForm(uri, form)
	if err != nil {
		return tokenResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tokenResponse{}, err
	}
	var out tokenResponse
	_ = json.Unmarshal(body, &out)
	if resp.StatusCode/100 != 2 {
		return out, fmt.Errorf("%w: %s %s", errRefresh, resp.Status, strings.TrimSpace(out.Error+" "+out.ErrorDesc))
	}
	if out.AccessToken == "" {
		return out, fmt.Errorf("%w: No access token in response.", errRefresh)
	}
	return out, nil
}

// refresh trades the refresh token for a new access token and updates
// t in place. A non-empty scopes is sent as the scope parameter and
// added to the recorded scopes when the endpoint accepts it.
func (t *Token) refresh(client *http.Client, now time.Time, scopes []string) error {
	if t.Refresh() == "" || t.str("client_id") == "" {
		return errRefresh
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {t.Refresh()},
		"client_id":     {t.str("client_id")},
		"client_secret": {t.str("client_secret")},
	}
	if len(scopes) > 0 {
		form.Set("scope", strings.Join(scopes, " "))
	}
	resp, err := postToken(client, t.URI(), form)
	if err != nil {
		return err
	}
	t.fields["token"] = resp.AccessToken
	if resp.RefreshToken != "" {
		t.fields["refresh_token"] = resp.RefreshToken
	}
	if len(scopes) > 0 {
		t.addScopes(scopes)
	}
	t.SetExpiry(resp.expiry(now))
	return nil
}

func (t *Token) addScopes(scopes []string) {
	have, _ := t.Scopes()
	seen := map[string]bool{}
	for _, s := range have {
		seen[s] = true
	}
	for _, s := range scopes {
		if !seen[s] {
			have = append(have, s)
			seen[s] = true
		}
	}
	t.fields["scopes"] = toAny(have)
}
