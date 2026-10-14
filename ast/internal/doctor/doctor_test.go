package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/agents"
	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestDoctorAgentsMissing(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	if err := config.Write(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	rep := Run(cfg, cfgPath, root, root, true)
	if rep.AgentsBlockReady {
		t.Fatal("should not be ready without AGENTS.md")
	}
	if rep.AgentsBlockStatus != "absent_file" {
		t.Fatalf("%s", rep.AgentsBlockStatus)
	}
	if rep.HookReady {
		t.Fatal("hook should not be ready without evidence")
	}
	if rep.HookStatus != "not_checked" {
		t.Fatalf("%s", rep.HookStatus)
	}
	if !rep.OK || !rep.Ready || rep.EnforcementStatus != "inactive" {
		t.Fatalf("unenforced checkout is inactive, not degraded: %+v", rep)
	}
}

func TestDoctorDegradedWhenEnforced(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfg.Enforcement.Mode = "always"
	cfgPath := filepath.Join(t.TempDir(), "c.toml")
	if err := config.Write(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	rep := Run(cfg, cfgPath, root, root, true)
	if rep.OK || rep.Ready || rep.EnforcementStatus != "degraded" {
		t.Fatalf("degraded: %+v", rep)
	}
	if !strings.Contains(FormatText(rep), "degraded: true") {
		t.Fatal(FormatText(rep))
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(agents.Block), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	configured := `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"rgw-ast hook"}]}]}}`
	if err := os.WriteFile(filepath.Join(root, ".claude", "settings.json"), []byte(configured), 0o644); err != nil {
		t.Fatal(err)
	}
	rep = Run(cfg, cfgPath, root, root, true)
	if !rep.OK || !rep.Ready || rep.EnforcementStatus != "ready" {
		t.Fatalf("ready: %+v notes=%v errors=%v", rep, rep.Notes, rep.Errors)
	}
}

func TestDoctorAgentsCurrent(t *testing.T) {
	root := t.TempDir()
	block := agents.Block
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(block), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfgPath := filepath.Join(t.TempDir(), "c.toml")
	_ = config.Write(cfgPath, cfg)
	rep := Run(cfg, cfgPath, root, root, true)
	if !rep.AgentsBlockReady || rep.AgentsBlockStatus != "current" {
		t.Fatalf("%+v", rep)
	}
}

func TestDoctorAgentsSuperficialBlockIsStale(t *testing.T) {
	root := t.TempDir()
	block := "<!-- rgw-ast:begin -->\nrgw-ast status --json\nrgw-ast exec\n<!-- rgw-ast:end -->\n"
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(block), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfgPath := filepath.Join(t.TempDir(), "c.toml")
	if err := config.Write(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	rep := Run(cfg, cfgPath, root, root, true)
	if rep.AgentsBlockReady || rep.AgentsBlockStatus != "stale" {
		t.Fatalf("superficial block must be stale: %+v", rep)
	}
}

func TestDoctorHookRequiresCommandField(t *testing.T) {
	root := t.TempDir()
	hookDir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(hookDir, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"note":"mentions rgw-ast hook but defines no hook"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfgPath := filepath.Join(t.TempDir(), "c.toml")
	if err := config.Write(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	rep := Run(cfg, cfgPath, root, root, true)
	if rep.HookReady || rep.HookStatus != "not_checked" {
		t.Fatalf("prose mention must not configure hook: %+v", rep)
	}

	configured := `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"rgw-ast hook"}]}]}}`
	if err := os.WriteFile(settings, []byte(configured), 0o644); err != nil {
		t.Fatal(err)
	}
	rep = Run(cfg, cfgPath, root, root, true)
	if !rep.HookReady || rep.HookStatus != "configured" {
		t.Fatalf("hook command not detected: %+v", rep)
	}
}

// TestMain isolates the user-level host settings doctor reads.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "doctor-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	os.Unsetenv("CLAUDE_CONFIG_DIR")
	os.Unsetenv("CODEX_HOME")
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

func TestDoctorUserLevelHookIsConfigured(t *testing.T) {
	root := t.TempDir()
	claude := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	cfgPath := filepath.Join(t.TempDir(), "c.toml")
	if err := config.Write(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	if rep := Run(cfg, cfgPath, root, root, true); rep.HookReady {
		t.Fatalf("no settings yet: %+v", rep)
	}
	configured := `{"hooks":{"PreToolUse":[{"matcher":"*","hooks":[{"type":"command","command":"rgw-ast hook","timeout":15}]}]}}`
	if err := os.WriteFile(filepath.Join(claude, "settings.json"), []byte(configured), 0o644); err != nil {
		t.Fatal(err)
	}
	rep := Run(cfg, cfgPath, root, root, true)
	if !rep.HookReady || rep.HookStatus != "configured" {
		t.Fatalf("user-level hook not detected: %+v", rep)
	}
}
