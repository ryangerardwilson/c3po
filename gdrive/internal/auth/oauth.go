package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DriveScope is full read and write access to Google Drive.
const DriveScope = "https://www.googleapis.com/auth/drive"

const expiryDelta = 10 * time.Second

// OAuthConfig is an installed-app OAuth 2.0 client.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	RedirectURL  string
	Scopes       []string
}

// Token is an OAuth 2.0 token.
type Token struct {
	AccessToken  string
	TokenType    string
	RefreshToken string
	Expiry       time.Time
}

// Valid reports whether the token has an access token that has not expired.
func (t *Token) Valid() bool {
	return t != nil && t.AccessToken != "" && !t.expired()
}

func (t *Token) expired() bool {
	if t.Expiry.IsZero() {
		return false
	}
	return t.Expiry.Round(0).Add(-expiryDelta).Before(time.Now())
}

// Type is the Authorization header scheme for the token.
func (t *Token) Type() string {
	switch strings.ToLower(t.TokenType) {
	case "", "bearer":
		return "Bearer"
	case "mac":
		return "MAC"
	case "basic":
		return "Basic"
	default:
		return t.TokenType
	}
}

type clientCredentials struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURIs []string `json:"redirect_uris"`
	AuthURI      string   `json:"auth_uri"`
	TokenURI     string   `json:"token_uri"`
}

// ConfigFromJSON reads a Google client secret file of the "web" or
// "installed" kind.
func ConfigFromJSON(data []byte, scopes ...string) (*OAuthConfig, error) {
	var file struct {
		Web       *clientCredentials `json:"web"`
		Installed *clientCredentials `json:"installed"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	creds := file.Web
	if creds == nil {
		creds = file.Installed
	}
	if creds == nil {
		return nil, errors.New("oauth2/google: no credentials found")
	}
	if len(creds.RedirectURIs) < 1 {
		return nil, errors.New("oauth2/google: missing redirect URL in the client_credentials.json")
	}
	return &OAuthConfig{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		AuthURL:      creds.AuthURI,
		TokenURL:     creds.TokenURI,
		RedirectURL:  creds.RedirectURIs[0],
		Scopes:       scopes,
	}, nil
}

// AuthCodeURL is the consent page URL. It asks for offline access and
// forces the consent prompt so a refresh token is returned.
func (c *OAuthConfig) AuthCodeURL(state string) string {
	values := url.Values{"response_type": {"code"}, "client_id": {c.ClientID}}
	if c.RedirectURL != "" {
		values.Set("redirect_uri", c.RedirectURL)
	}
	if len(c.Scopes) > 0 {
		values.Set("scope", strings.Join(c.Scopes, " "))
	}
	values.Set("state", state)
	values.Set("access_type", "offline")
	values.Set("prompt", "consent")
	sep := "?"
	if strings.Contains(c.AuthURL, "?") {
		sep = "&"
	}
	return c.AuthURL + sep + values.Encode()
}

// Exchange trades an authorization code for a token.
func (c *OAuthConfig) Exchange(ctx context.Context, code string) (*Token, error) {
	values := url.Values{"grant_type": {"authorization_code"}, "code": {code}}
	if c.RedirectURL != "" {
		values.Set("redirect_uri", c.RedirectURL)
	}
	return c.retrieveToken(ctx, values)
}

func (c *OAuthConfig) refresh(ctx context.Context, refreshToken string) (*Token, error) {
	values := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}}
	token, err := c.retrieveToken(ctx, values)
	if err != nil {
		return nil, err
	}
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}
	return token, nil
}

func (c *OAuthConfig) retrieveToken(ctx context.Context, values url.Values) (*Token, error) {
	values.Set("client_id", c.ClientID)
	if c.ClientSecret != "" {
		values.Set("client_secret", c.ClientSecret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.TokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth2: cannot fetch token: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("oauth2: cannot fetch token: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, tokenError(resp.Status, body)
	}
	return parseTokenResponse(body)
}

func parseTokenResponse(body []byte) (*Token, error) {
	var reply struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &reply); err != nil {
		return nil, fmt.Errorf("oauth2: cannot parse json: %v", err)
	}
	if reply.AccessToken == "" {
		return nil, errors.New("oauth2: server response missing access_token")
	}
	token := &Token{AccessToken: reply.AccessToken, TokenType: reply.TokenType, RefreshToken: reply.RefreshToken}
	if reply.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(reply.ExpiresIn) * time.Second)
	}
	return token, nil
}

func tokenError(status string, body []byte) error {
	var reply struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		ErrorURI         string `json:"error_uri"`
	}
	if json.Unmarshal(body, &reply) != nil || reply.Error == "" {
		return fmt.Errorf("oauth2: cannot fetch token: %v\nResponse: %s", status, body)
	}
	text := fmt.Sprintf("oauth2: %q", reply.Error)
	if reply.ErrorDescription != "" {
		text += fmt.Sprintf(" %q", reply.ErrorDescription)
	}
	if reply.ErrorURI != "" {
		text += fmt.Sprintf(" %q", reply.ErrorURI)
	}
	return errors.New(text)
}

// TokenSource returns a valid token and refreshes it when it expires.
type TokenSource struct {
	ctx   context.Context
	conf  *OAuthConfig
	mu    sync.Mutex
	token *Token
}

// NewTokenSource starts from token and refreshes with conf.
func NewTokenSource(ctx context.Context, conf *OAuthConfig, token *Token) *TokenSource {
	return &TokenSource{ctx: ctx, conf: conf, token: token}
}

// Token returns the current token, refreshing it first when needed.
func (s *TokenSource) Token() (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token.Valid() {
		return s.token, nil
	}
	if s.token == nil || s.token.RefreshToken == "" {
		return nil, errors.New("oauth2: token expired and refresh token is not set")
	}
	refreshed, err := s.conf.refresh(s.ctx, s.token.RefreshToken)
	if err != nil {
		return nil, err
	}
	s.token = refreshed
	return refreshed, nil
}

type transport struct {
	source *TokenSource
	base   http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.source.Token()
	if err != nil {
		if req.Body != nil {
			_ = req.Body.Close()
		}
		return nil, err
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", token.Type()+" "+token.AccessToken)
	return t.base.RoundTrip(clone)
}

// NewClient is an HTTP client that authorizes each request with source.
func NewClient(source *TokenSource) *http.Client {
	return &http.Client{Transport: &transport{source: source, base: http.DefaultTransport}}
}
