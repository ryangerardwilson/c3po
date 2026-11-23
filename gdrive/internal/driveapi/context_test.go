package driveapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestARequestEndsWithItsContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	client := newClient(server.Client(), server.URL+"/", server.URL+"/upload/")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := client.WithContext(ctx).ListChildren("root"); err == nil {
		t.Fatal("a stuck list returned")
	}
	if took := time.Since(start); took > time.Second {
		t.Fatalf("the list took %v", took)
	}
	if client.ctx != nil {
		t.Fatal("WithContext changed the client it was called on")
	}
}
