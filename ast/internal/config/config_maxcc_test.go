package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaxCCDefault(t *testing.T) {
	for body, want := range map[string]int{
		"version = 1\n":              DefaultMaxCC,
		"version = 1\nmax_cc = 0\n":  DefaultMaxCC,
		"version = 1\nmax_cc = -4\n": DefaultMaxCC,
		"version = 1\nmax_cc = 12\n": 12,
	} {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		cfg, err := Read(path)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.MaxCC != want {
			t.Errorf("%q: max_cc %d, want %d", body, cfg.MaxCC, want)
		}
		after, _ := os.ReadFile(path)
		if string(after) != body {
			t.Errorf("%q: Read rewrote the file to %q", body, after)
		}
	}
	if DefaultMaxCC != 30 || Default().MaxCC != 30 {
		t.Fatal("max_cc default must be 30")
	}
}
