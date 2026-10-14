package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultExcludes(t *testing.T) {
	cfg := Default()
	want := []string{
		"**/node_modules/**",
		"**/vendor/**",
		"**/.next/**",
		"**/.serverless/**",
		"**/*_test.go",
		"**/test_*.py",
		"**/*.test.*",
		"**/*.spec.*",
		"**/tests/**",
		"**/test/**",
	}
	have := append(append([]string{}, cfg.Exclude.Patterns...), cfg.Exclude.Tests...)
	for _, w := range want {
		if !contains(have, w) {
			t.Errorf("missing exclude %s", w)
		}
	}
	if cfg.Thresholds.Cognitive != 15 || cfg.Thresholds.Lines != 80 || cfg.Thresholds.Params != 6 {
		t.Fatalf("thresholds %+v", cfg.Thresholds)
	}
	if cfg.Thresholds.Cyclomatic != 10 {
		t.Fatalf("cyclomatic %d", cfg.Thresholds.Cyclomatic)
	}
	if cfg.Thresholds.Models.Degree != 10 || cfg.Conventions.Naming != true {
		t.Fatalf("models or conventions %+v %+v", cfg.Thresholds.Models, cfg.Conventions)
	}
}

func TestLoadOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "complexity.toml")
	body := "version = 1\n[thresholds]\ncognitive = 3\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path, Default())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Thresholds.Cognitive != 3 || cfg.Thresholds.Lines != 80 {
		t.Fatalf("merge cognitive=%d lines=%d", cfg.Thresholds.Cognitive, cfg.Thresholds.Lines)
	}
}

func TestFindProjectStopsAtGitRoot(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "a")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(root), "complexity.toml")
	_ = outside
	if err := os.WriteFile(filepath.Join(root, "complexity.toml"), []byte("version = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := FindProject(child, root)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(root, "complexity.toml") {
		t.Fatalf("got %s", got)
	}
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
