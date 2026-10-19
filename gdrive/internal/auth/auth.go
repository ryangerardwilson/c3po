// Package auth runs the Google OAuth flow and keeps per-account tokens.
package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ryangerardwilson/c3po/gdrive/internal/config"
	"github.com/ryangerardwilson/c3po/gdrive/internal/googleapi"
	"github.com/ryangerardwilson/c3po/gdrive/internal/paths"
)

const localServerTimeout = 60 * time.Second

// aboutURL is the Drive about endpoint. Tests point it at a local server.
var aboutURL = googleapi.DriveBase + "about"

type storedToken struct {
	AccessToken  string    `json:"access_token,omitempty"`
	Token        string    `json:"token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

// AuthorizeAccount runs the browser consent flow for a client secret file,
// stores the token under the account email, and returns both.
func AuthorizeAccount(clientSecretFile string, stdin io.Reader, stdout, stderr io.Writer) (*Token, string, error) {
	if err := paths.EnsureDirs(); err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(paths.ExpandHome(clientSecretFile))
	if err != nil {
		return nil, "", fmt.Errorf("oauth authorization failed: %w", err)
	}
	oauthConfig, err := ConfigFromJSON(data, DriveScope)
	if err != nil {
		return nil, "", fmt.Errorf("oauth authorization failed: %w", err)
	}
	token, err := completeOAuth(oauthConfig, stdin, stdout, stderr)
	if err != nil {
		return nil, "", err
	}
	ctx := context.Background()
	email, err := LookupEmail(ctx, NewClient(NewTokenSource(ctx, oauthConfig, token)))
	if err != nil {
		return nil, "", err
	}
	if email == "" {
		return nil, "", fmt.Errorf("drive profile lookup returned no email address")
	}
	if err := WriteToken(paths.TokenFileForEmail(email), token); err != nil {
		return nil, "", err
	}
	return token, email, nil
}

// HTTPClient is an authorized client for the account. It refreshes the
// stored token and writes it back when the refresh changed it.
func HTTPClient(ctx context.Context, account *config.Account) (*http.Client, error) {
	if account.Email == "" {
		return nil, fmt.Errorf("preset %s is missing email; re-run `c3po gdrive auth <client_secret_path>`", account.Preset)
	}
	secret, oauthConfig, err := accountOAuthConfig(account)
	if err != nil {
		return nil, err
	}
	tokenPath := paths.TokenFileForEmail(account.Email)
	token, err := ReadToken(tokenPath)
	if err != nil {
		return nil, err
	}
	if !token.Valid() && token.RefreshToken == "" {
		token, _, err = AuthorizeAccount(secret, os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
			return nil, err
		}
	}
	source := NewTokenSource(ctx, oauthConfig, token)
	refreshed, err := source.Token()
	if err != nil {
		return nil, fmt.Errorf("oauth refresh failed: %w", err)
	}
	if tokenChanged(token, refreshed) {
		_ = WriteToken(tokenPath, refreshed)
	}
	return NewClient(source), nil
}

func accountOAuthConfig(account *config.Account) (string, *OAuthConfig, error) {
	secret, err := config.RequireClientSecret(account)
	if err != nil {
		return "", nil, err
	}
	data, err := os.ReadFile(paths.ExpandHome(secret))
	if err != nil {
		return "", nil, err
	}
	oauthConfig, err := ConfigFromJSON(data, DriveScope)
	if err != nil {
		return "", nil, err
	}
	return secret, oauthConfig, nil
}

func tokenChanged(before, after *Token) bool {
	return after.AccessToken != before.AccessToken ||
		after.RefreshToken != before.RefreshToken ||
		!after.Expiry.Equal(before.Expiry)
}

// LookupEmail asks Drive for the email address of the authorized user.
func LookupEmail(ctx context.Context, client *http.Client) (string, error) {
	query := url.Values{"alt": {"json"}, "fields": {"user(emailAddress)"}, "prettyPrint": {"false"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, aboutURL+"?"+query.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("drive profile lookup failed after oauth: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("drive profile lookup failed after oauth: %w", err)
	}
	defer resp.Body.Close()
	if err := googleapi.CheckResponse(resp); err != nil {
		return "", fmt.Errorf("drive profile lookup failed after oauth: %w", err)
	}
	var about struct {
		User struct {
			EmailAddress string `json:"emailAddress"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&about); err != nil {
		return "", fmt.Errorf("drive profile lookup failed after oauth: %w", err)
	}
	return config.NormalizeEmail(about.User.EmailAddress), nil
}

// ReadToken reads a token file. It accepts the "token" key the Python app
// wrote as well as "access_token".
func ReadToken(path string) (*Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw storedToken
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	access := raw.AccessToken
	if access == "" {
		access = raw.Token
	}
	tokenType := raw.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}
	return &Token{
		AccessToken:  access,
		TokenType:    tokenType,
		RefreshToken: raw.RefreshToken,
		Expiry:       raw.Expiry,
	}, nil
}

// WriteToken writes a token file readable only by the user.
func WriteToken(path string, token *Token) error {
	raw := storedToken{
		AccessToken:  token.AccessToken,
		Token:        token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func completeOAuth(oauthConfig *OAuthConfig, stdin io.Reader, stdout, stderr io.Writer) (*Token, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("oauth local server failed: %w", err)
	}
	defer listener.Close()
	oauthConfig.RedirectURL = "http://" + listener.Addr().String() + "/"
	state := fmt.Sprintf("gdrive-%d", time.Now().UnixNano())
	authURL := oauthConfig.AuthCodeURL(state)
	fmt.Fprintf(stdout, "Please visit this URL to authorize this application:\n%s\n\n", authURL)
	_ = exec.Command("xdg-open", authURL).Start()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	server := &http.Server{Handler: callbackHandler(state, codeCh, errCh)}
	go func() { _ = server.Serve(listener) }()
	defer server.Shutdown(context.Background())

	select {
	case code := <-codeCh:
		return oauthConfig.Exchange(context.Background(), code)
	case err := <-errCh:
		return nil, err
	case <-time.After(localServerTimeout):
		fmt.Fprintln(stderr, "No browser callback was received. Paste the callback URL or code below.")
		code, err := promptForCode(stdin, stdout)
		if err != nil {
			return nil, err
		}
		return oauthConfig.Exchange(context.Background(), code)
	}
}

func callbackHandler(state string, codeCh chan<- string, errCh chan<- error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			sendErr(errCh, fmt.Errorf("oauth callback state mismatch"))
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			sendErr(errCh, fmt.Errorf("oauth callback URL did not contain a code"))
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		fmt.Fprintln(w, "gdrive authorization complete. Return to the terminal.")
		select {
		case codeCh <- code:
		default:
		}
	})
}

func sendErr(errCh chan<- error, err error) {
	select {
	case errCh <- err:
	default:
	}
}

func promptForCode(stdin io.Reader, stdout io.Writer) (string, error) {
	fmt.Fprint(stdout, "Paste the full localhost callback URL from the browser address bar, or just the code value: ")
	scanner := bufio.NewScanner(stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("oauth callback was empty")
	}
	value := strings.Trim(strings.TrimSpace(scanner.Text()), "'\"`;")
	if value == "" {
		return "", fmt.Errorf("oauth callback was empty")
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		return value, nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	code := parsed.Query().Get("code")
	if code == "" {
		return "", fmt.Errorf("oauth callback URL did not contain a code")
	}
	return code, nil
}
