package enforce

import "github.com/ryangerardwilson/c3po/ast/internal/config"

// Decision is the enforcement outcome for a workspace.
type Decision struct {
	Mode         string
	ThresholdLOC int
	LOC          int
	Enforced     bool
}

// Decide applies global policy to measured LOC.
func Decide(cfg config.Config, loc int) Decision {
	d := Decision{
		Mode:         cfg.Enforcement.Mode,
		ThresholdLOC: cfg.ThresholdLOC,
		LOC:          loc,
	}
	switch cfg.Enforcement.Mode {
	case "always":
		d.Enforced = true
	case "never":
		d.Enforced = false
	default: // auto
		d.Enforced = loc >= cfg.ThresholdLOC
	}
	return d
}
