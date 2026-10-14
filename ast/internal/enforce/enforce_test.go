package enforce

import (
	"testing"

	"github.com/ryangerardwilson/c3po/ast/internal/config"
)

func TestDecide(t *testing.T) {
	cfg := config.Default()
	cfg.ThresholdLOC = 5000
	cfg.Enforcement.Mode = "auto"
	if Decide(cfg, 4999).Enforced {
		t.Fatal("below")
	}
	if !Decide(cfg, 5000).Enforced {
		t.Fatal("at threshold")
	}
	cfg.Enforcement.Mode = "always"
	if !Decide(cfg, 0).Enforced {
		t.Fatal("always")
	}
	cfg.Enforcement.Mode = "never"
	if Decide(cfg, 1_000_000).Enforced {
		t.Fatal("never")
	}
}
