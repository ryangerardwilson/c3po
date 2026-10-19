package dns

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecide(t *testing.T) {
	id := "1bdc9e04-7e31-4589-baa9-f6caf6ae9a74"
	target := Target(id)
	if Decide("notes.ryangerardwilson.com", id, nil) != "create" {
		t.Fatal("missing record should be created")
	}
	keep := []Record{{Type: "CNAME", Name: "notes.ryangerardwilson.com", Content: target}}
	if Decide("notes.ryangerardwilson.com", id, keep) != "keep" {
		t.Fatal("matching cname should be kept")
	}
	foreign := []Record{{Type: "CNAME", Name: "notes.ryangerardwilson.com", Content: "elsewhere.example"}}
	if Decide("notes.ryangerardwilson.com", id, foreign) != "refuse" {
		t.Fatal("foreign cname should be refused")
	}
	mixed := []Record{
		{Type: "CNAME", Content: target},
		{Type: "A", Content: "192.0.2.1"},
	}
	if Decide("notes.ryangerardwilson.com", id, mixed) != "refuse" {
		t.Fatal("mixed records should be refused")
	}
}

func TestEnsureLeavesForeignDNS(t *testing.T) {
	id := "1bdc9e04-7e31-4589-baa9-f6caf6ae9a74"
	var created bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("missing authorization")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/zones":
			io.WriteString(w, `{"success":true,"result":[{"id":"zone1"}]}`)
		case strings.Contains(r.URL.Path, "/dns_records") && r.Method == http.MethodGet:
			io.WriteString(w, `{"success":true,"result":[{"type":"CNAME","name":"notes.ryangerardwilson.com","content":"elsewhere.example"}]}`)
		default:
			created = true
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	}))
	defer srv.Close()
	client := &Client{Token: "test-token", BaseURL: srv.URL}
	err := client.Ensure(context.Background(), "ryangerardwilson.com", "notes.ryangerardwilson.com", id)
	if !errors.Is(err, ErrForeign) {
		t.Fatalf("err %v", err)
	}
	if created {
		t.Fatal("foreign name was written")
	}
}

func TestEnsureCreatesProxiedCNAME(t *testing.T) {
	id := "1bdc9e04-7e31-4589-baa9-f6caf6ae9a74"
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/zones":
			io.WriteString(w, `{"success":true,"result":[{"id":"zone1"}]}`)
		case r.Method == http.MethodGet:
			io.WriteString(w, `{"success":true,"result":[]}`)
		case r.Method == http.MethodPost:
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			io.WriteString(w, `{"success":true,"result":{"id":"rec"}}`)
		}
	}))
	defer srv.Close()
	client := &Client{Token: "test-token", BaseURL: srv.URL}
	if err := client.Ensure(context.Background(), "ryangerardwilson.com", "lab.ryangerardwilson.com", id); err != nil {
		t.Fatal(err)
	}
	if body["type"] != "CNAME" || body["name"] != "lab.ryangerardwilson.com" || body["proxied"] != true {
		t.Fatalf("body %+v", body)
	}
	if body["content"] != Target(id) {
		t.Fatalf("content %v", body["content"])
	}
}
