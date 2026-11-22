package sqlxdb

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordRoundTrip(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SQLX_ROOT", root)
	path := filepath.Join(t.TempDir(), "c3po", "gmail", "tokens", "user.json")
	body := []byte(`{"refresh_token":"it's fine"}`)
	if err := WriteToken("gmail", path, body); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("production write created a file: %v", err)
	}
	got, err := ReadToken("gmail", path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Fatalf("body = %q", got)
	}
	if _, err := ReadToken("gcal", path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("another app's token: %v", err)
	}
	if err := WriteToken("gmail", path, []byte("not json")); err == nil {
		t.Fatal("a token that is not a JSON object was stored")
	}
	info, err := os.Stat(filepath.Join(root, "data", "c3po.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(root, "orm", "c3po")); err != nil {
		t.Fatal(err)
	}
}

func TestOutsideLoginHomeStaysAFile(t *testing.T) {
	t.Setenv("SQLX_ROOT", "")
	t.Setenv("C3PO_SQLX", "")
	path := filepath.Join(t.TempDir(), "c3po", "gmail", "tokens", "user.json")
	body := []byte("local")
	if err := WriteToken("gmail", path, body); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "local" {
		t.Fatalf("file = %q", got)
	}
	ok, err := TokenExists("gmail", path)
	if err != nil || !ok {
		t.Fatalf("exists %v %v", ok, err)
	}
}

func TestFilePathStaysAFile(t *testing.T) {
	t.Setenv("SQLX_ROOT", "")
	t.Setenv("C3PO_SQLX", "off")
	path := filepath.Join(t.TempDir(), "token.json")
	body := []byte("local")
	if err := WriteToken("gmail", path, body); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "local" {
		t.Fatalf("file = %q", got)
	}
}
