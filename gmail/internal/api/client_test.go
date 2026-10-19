package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListIDsAndTrash(t *testing.T) {
	var trashed string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("auth %s", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/messages":
			w.Write([]byte(`{"messages":[{"id":"a"},{"id":"b"}],"nextPageToken":""}`))
		case "/messages/batchModify":
			trashed = r.URL.Path
			w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), Base: srv.URL, Access: func() (string, error) { return "tok", nil }}
	ids, err := c.ListIDs("from:clickup.com")
	if err != nil || len(ids) != 2 || ids[0] != "a" {
		t.Fatalf("%v %v", ids, err)
	}
	n, err := c.TrashIDs(ids)
	if err != nil || n != 2 || trashed == "" {
		t.Fatalf("trash %d %v %s", n, err, trashed)
	}
}
