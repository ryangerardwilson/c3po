package store

import (
	"context"
	"errors"
	"testing"

	"github.com/ryangerardwilson/c3po/tunnel/internal/routes"
	"github.com/ryangerardwilson/c3po/tunnel/internal/secrets"
)

func TestAddRefusedNameIsNotSaved(t *testing.T) {
	home := t.TempDir()
	load := func() (secrets.Tunnel, error) { return secrets.Tunnel{ID: "id", Zone: routes.Zone}, nil }
	refuse := func(context.Context, string, string, string) error { return errors.New("refused") }
	if _, _, err := Add(home, "lab", "9090", load, refuse); err == nil {
		t.Fatal("refused add succeeded")
	}
	file, err := routes.Load(routes.Path(home))
	if err != nil || len(file.Routes) != 1 || file.Routes[0].Name != routes.Apex {
		t.Fatalf("file %+v %v", file, err)
	}
	if _, _, err := Add(home, "lab", "9090", nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := Remove(home, "lab"); err != nil {
		t.Fatal(err)
	}
	if _, err := Remove(home, "lab"); err == nil {
		t.Fatal("missing route was removed")
	}
	if _, _, err := Add(home, "lab", "0", nil, nil); err == nil {
		t.Fatal("port 0 was accepted")
	}
}
