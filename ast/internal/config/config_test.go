package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureAndLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected missing config, got %v", err)
	}
	cfg, gotPath, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != path {
		t.Fatalf("path %s != %s", gotPath, path)
	}
	if cfg.ThresholdLOC != DefaultThresholdLOC {
		t.Fatalf("threshold %d", cfg.ThresholdLOC)
	}
	if cfg.Enforcement.Mode != "auto" {
		t.Fatalf("mode %s", cfg.Enforcement.Mode)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	// second load reads existing
	cfg2, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.ThresholdLOC != cfg.ThresholdLOC {
		t.Fatal("reload mismatch")
	}
}

func TestReadInvalidMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("threshold_loc = 10\n[enforcement]\nmode = \"nope\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Fatal("expected error")
	}
}
