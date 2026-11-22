package xai

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/idealake/internal/lake"
)

func TestEmbedRemote(t *testing.T) {
	var gotAuth, gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var body embedRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		gotModel = body.Model
		if body.EncodingFormat != "float" || len(body.Input) != 2 {
			t.Errorf("body %#v", body)
		}
		_, _ = io.WriteString(w, `{"data":[{"index":1,"embedding":[0,1]},{"index":0,"embedding":[1,0]}]}`)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("XAI_API_KEY", "test-key")
	vectors, err := Client{URL: srv.URL}.Embed([]string{"passage: a", "passage: b"})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer test-key" || gotModel != lake.EmbedModel {
		t.Fatalf("auth %s model %s", gotAuth, gotModel)
	}
	if len(vectors) != 2 || vectors[0][0] != 1 || vectors[1][1] != 1 {
		t.Fatalf("vectors %#v", vectors)
	}
}

func TestEmbedRemoteHidesTheKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "bad "+r.Header.Get("Authorization"))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("XAI_API_KEY", "test-key")
	_, err := Client{URL: srv.URL}.Embed([]string{"passage: a"})
	if err == nil || strings.Contains(err.Error(), "test-key") || strings.Contains(err.Error(), "Bearer") {
		t.Fatal(err)
	}
}

func TestEmbedRemoteNeedsKey(t *testing.T) {
	t.Setenv("XAI_API_KEY", "")
	_, err := Client{URL: URL}.Embed([]string{"passage: a"})
	if err == nil || strings.Contains(err.Error(), "http") {
		t.Fatal(err)
	}
}
