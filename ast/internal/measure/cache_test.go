package measure

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestCountCached(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.go"), "package a\n")
	cfg := config.Default()
	cfg.Cache.Dir = t.TempDir()

	start := time.Now()
	r1, err := CountCached(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if r1.LOC < 1 {
		t.Fatal(r1)
	}
	// second should hit cache quickly
	r2, err := CountCached(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if r2.LOC != r1.LOC {
		t.Fatalf("%d vs %d", r2.LOC, r1.LOC)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("too slow")
	}
}
