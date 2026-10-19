package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const successPage = "The authentication flow has completed. You may close this window."

// clientConfig is the "installed" or "web" block of a client secret.
type clientConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AuthURI      string `json:"auth_uri"`
	TokenURI     string `json:"token_uri"`
}

func readClientSecret(path string) (clientConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return clientConfig{}, err
	}
	var file map[string]clientConfig
	if err := json.Unmarshal(raw, &file); err != nil {
		return clientConfig{}, fmt.Errorf("invalid client secret file %s: %w", path, err)
	}
	cfg, ok := file["web"]
	if !ok {
		cfg, ok = file["installed"]
	}
	if !ok {
		return clientConfig{}, errors.New("Client secrets must be for a web or installed app.")
	}
	if cfg.ClientID == "" || cfg.AuthURI == "" || cfg.TokenURI == "" {
		return clientConfig{}, errors.New("Client secrets is not in the correct format.")
	}
	return cfg, nil
}

// grant is the code the loopback redirect delivered.
type grant struct {
	code string
	err  error
}

// runFlow runs the installed-app flow on a loopback port and returns
// the new token, like google-auth-oauthlib run_local_server(port=0).
func (o Options) runFlow(cfg clientConfig, scopes []string) (*Token, error) {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}
	redirect := fmt.Sprintf("http://localhost:%d/", ln.Addr().(*net.TCPAddr).Port)
	state, verifier := randomString(30), randomString(128)
	authURL := buildAuthURL(cfg, scopes, redirect, state, verifier)
	got := make(chan grant, 1)
	srv := &http.Server{Handler: callback(state, got)}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Shutdown(context.Background())
	fmt.Fprintf(o.Out, "Please visit this URL to authorize this application: %s\n", authURL)
	if o.OpenBrowser != nil {
		_ = o.OpenBrowser(authURL)
	}
	g := <-got
	if g.err != nil {
		return nil, g.err
	}
	return o.exchange(cfg, scopes, redirect, g.code, verifier)
}

func buildAuthURL(cfg clientConfig, scopes []string, redirect, state, verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {cfg.ClientID},
		"redirect_uri":          {redirect},
		"scope":                 {strings.Join(scopes, " ")},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
		"access_type":           {"offline"},
	}
	sep := "?"
	if strings.Contains(cfg.AuthURI, "?") {
		sep = "&"
	}
	return cfg.AuthURI + sep + q.Encode()
}

// callback answers the browser redirect. Requests without a code or an
// error, like a favicon fetch, are ignored.
func callback(state string, got chan<- grant) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		g := grant{code: q.Get("code")}
		switch {
		case q.Get("error") != "":
			g.err = fmt.Errorf("authorization failed: %s", q.Get("error"))
		case g.code == "":
			http.NotFound(w, r)
			return
		case q.Get("state") != state:
			g.err = errors.New("mismatching_state: CSRF Warning! State not equal in request and response.")
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, successPage)
		select {
		case got <- g:
		default:
		}
	})
}

func (o Options) exchange(cfg clientConfig, scopes []string, redirect, code, verifier string) (*Token, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirect},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code_verifier": {verifier},
	}
	resp, err := postToken(o.client(), cfg.TokenURI, form)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	exp := resp.expiry(o.now())
	return NewToken(resp.AccessToken, resp.RefreshToken, cfg.TokenURI, cfg.ClientID, cfg.ClientSecret, scopes, exp), nil
}

const verifierChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"

func randomString(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	for i, b := range buf {
		buf[i] = verifierChars[int(b)%len(verifierChars)]
	}
	return string(buf)
}
