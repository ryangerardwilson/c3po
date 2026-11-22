package slackapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/slack/internal/app"
)

func TestRefusedCallIsAnAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/conversations.mark":
			_, _ = w.Write([]byte(`{"ok":false,"error":"missing_scope"}`))
		case "/api/conversations.close":
			_, _ = w.Write([]byte(`{"ok":false}`))
		default:
			t.Errorf("path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := Client{Token: "xoxp-user", HTTPClient: server.Client(), BaseURL: server.URL + "/api/"}
	err := client.MarkRead("G1", "1.000100")
	var refused *app.APIError
	if !errors.As(err, &refused) || refused.Code != "missing_scope" || err.Error() != "Slack API conversations.mark failed: missing_scope" {
		t.Fatalf("mark error %#v", err)
	}
	err = client.CloseConversation("D1")
	if !errors.As(err, &refused) || err.Error() != "Slack API conversations.close failed: unknown_error" {
		t.Fatalf("close error %#v", err)
	}
}

func TestConnectBoundsCallsAndKeepsTransport(t *testing.T) {
	transport := &http.Transport{}
	connector := Connector{HTTPClient: &http.Client{Timeout: time.Minute, Transport: transport}, BaseURL: "http://example.invalid/api/"}
	bounded := connector.Connect(app.ConnectOptions{Token: "xoxp-user", Timeout: 45 * time.Second}).(Client)
	if bounded.HTTPClient.Timeout != 45*time.Second || bounded.HTTPClient.Transport != transport || bounded.Token != "xoxp-user" {
		t.Fatalf("bounded %#v", bounded.HTTPClient)
	}
	plain := connector.Connect(app.ConnectOptions{Token: "xoxp-user"}).(Client)
	if plain.HTTPClient != connector.HTTPClient {
		t.Fatalf("plain %#v", plain.HTTPClient)
	}
}
