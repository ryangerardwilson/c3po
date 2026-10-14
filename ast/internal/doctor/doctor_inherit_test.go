package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/agents"
	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestDoctorInheritsAncestorBlock(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "repo")
	if err := os.MkdirAll(filepath.Join(child, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "AGENTS.md"), []byte("# Parent\n\n"+agents.Block), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "AGENTS.md"), []byte("# Repo rules only\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()
	rep := Run(cfg, filepath.Join(t.TempDir(), "config.toml"), child, "", true)
	if rep.AgentsBlockStatus != "inherited" || !rep.AgentsBlockReady {
		t.Fatalf("status %s ready %v", rep.AgentsBlockStatus, rep.AgentsBlockReady)
	}

	if err := os.WriteFile(filepath.Join(parent, "AGENTS.md"), []byte("<!-- rgw-ast:begin -->\nold\n<!-- rgw-ast:end -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep = Run(cfg, filepath.Join(t.TempDir(), "config.toml"), child, "", true)
	if rep.AgentsBlockStatus != "missing_block" || rep.AgentsBlockReady {
		t.Fatalf("stale ancestor must not count: %s", rep.AgentsBlockStatus)
	}
}

func TestCanonicalBlockIsShort(t *testing.T) {
	lines := 0
	for _, c := range agents.Block {
		if c == '\n' {
			lines++
		}
	}
	if lines > 30 {
		t.Fatalf("agents block has %d lines; keep it at most 30", lines)
	}
}
