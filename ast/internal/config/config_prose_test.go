package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOmittedProseKeysUseDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("version = 1\nthreshold_loc = 5000\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Prose) == 0 || cfg.Prose[0] != "**/*.md" || cfg.Enforcement.CountProse {
		t.Fatalf("prose %v count %v", cfg.Prose, cfg.Enforcement.CountProse)
	}
}
