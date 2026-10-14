package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGlobalPathUsesXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-example")
	got, err := GlobalPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/tmp/xdg-example", "complexity", "settings.json")
	if got != want {
		t.Fatalf("got %s", got)
	}
}

func TestGlobalPathDefaultsToHomeConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	got, err := GlobalPath()
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "complexity", "settings.json")
	if got != want {
		t.Fatalf("got %s", got)
	}
}

func TestGlobalJSONKeepsAbsentKeys(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, "complexity")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("{\"thresholds\":{\"cognitive\":4}}\n")
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Merge(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Thresholds.Cognitive != 4 || cfg.Thresholds.Lines != 80 || cfg.Thresholds.Params != 6 {
		t.Fatalf("merged %+v", cfg.Thresholds)
	}
	if len(cfg.Exclude.Patterns) == 0 {
		t.Fatal("exclude patterns were cleared")
	}
}

func TestProjectTomlOverridesGlobalJSON(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, "complexity")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("{\"thresholds\":{\"cognitive\":4}}\n")
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	proj := "version = 1\n[thresholds]\ncognitive = 9\n"
	if err := os.WriteFile(filepath.Join(root, "complexity.toml"), []byte(proj), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Merge(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Thresholds.Cognitive != 9 || cfg.Thresholds.Lines != 80 {
		t.Fatalf("merged %+v", cfg.Thresholds)
	}
}

func TestEnsureWritesSettingsJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := GlobalPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := Ensure(path); err != nil {
		t.Fatal(err)
	}
	if err := Ensure(path); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path, Default())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Thresholds.Cognitive != 15 || cfg.Enforcement.Mode != "auto" {
		t.Fatalf("%+v", cfg)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() < 20 {
		t.Fatalf("short file %d", info.Size())
	}
}

func TestRejectsBadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, Default()); err == nil {
		t.Fatal("expected error")
	}
}
