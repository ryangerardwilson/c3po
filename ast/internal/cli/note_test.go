package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestPatchPrintsTheHash(t *testing.T) {
	ws := t.TempDir()
	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	if err := config.Write(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(ws, "f.go"), "package p\nconst X = 1\n")
	var out, errB bytes.Buffer
	r := NewRunner(&out, &errB)
	r.Cwd = ws
	r.ConfigPath = cfgPath
	if code := r.Run([]string{"hash", "f.go"}); code != 0 {
		t.Fatal(code, errB.String())
	}
	hash := firstField(out.String())
	out.Reset()
	if code := r.Run([]string{"patch", "f.go", "--expect-hash", hash, "--old", "X = 1", "--new", "X = 2"}); code != 0 {
		t.Fatal(code, errB.String(), out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("ok  ")) || bytes.Contains(out.Bytes(), []byte("cognitive")) {
		t.Fatalf("%s", out.String())
	}
}

func firstField(s string) string {
	for i, c := range s {
		if c == ' ' || c == '\n' {
			return s[:i]
		}
	}
	return s
}
