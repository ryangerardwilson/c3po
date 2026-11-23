package auth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// stuckTokens is a token endpoint that never answers. It reads the form
// first, so the server notices when the caller hangs up, and the test lets
// it go before closing the server.
func stuckTokens(t *testing.T) *OAuthConfig {
	t.Helper()
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(func() {
		close(release)
		server.Close()
	})
	return &OAuthConfig{ClientID: "id", TokenURL: server.URL}
}

func TestARefreshEndsWithTheRequestContext(t *testing.T) {
	client := NewClient(NewTokenSource(context.Background(), stuckTokens(t), &Token{RefreshToken: "r"}))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:1/never", nil)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := client.Do(req); err == nil {
		t.Fatal("a stuck refresh returned a response")
	}
	if took := time.Since(start); took > time.Second {
		t.Fatalf("the refresh took %v", took)
	}
}

func TestLoginClientRefreshEndsWithItsContext(t *testing.T) {
	token := &Token{RefreshToken: "r"}
	login := &Login{token: token, source: NewTokenSource(context.Background(), stuckTokens(t), token)}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := login.Client(ctx); err == nil {
		t.Fatal("a stuck refresh returned a client")
	}
	if took := time.Since(start); took > time.Second {
		t.Fatalf("the refresh took %v", took)
	}
}
