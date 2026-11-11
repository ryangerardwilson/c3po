// Package usage records which c3po feature ran, and whether a human, an agent,
// or a service ran it. The record is a daily counter. It does not store
// arguments, paths, presets, or message text.
package usage

import "strings"

// Catalog is every feature whose absence is evidence. A feature with no row
// is unused. A name that is not in this list is reported apart from it, so a
// missed command stays visible without becoming a new feature by being typed.
var Catalog = []string{
	"c3po",
	"help",
	"version",
	"usage",

	"tui",
	"tui cyclotree",
	"tui l",
	"tui o",
	"tui tunnel",
	"tui gdrive",
	"tui pass",
	"tui deport",
	"tui xl",
	"tui looptab",

	"ast config",
	"ast status",
	"ast measure",
	"ast map",
	"ast show",
	"ast search",
	"ast callers",
	"ast callees",
	"ast impact",
	"ast explain",
	"ast hash",
	"ast read",
	"ast create",
	"ast append",
	"ast patch",
	"ast apply",
	"ast delete",
	"ast move",
	"ast exec",
	"ast doctor",
	"ast agents-block",
	"ast hook",
	"ast packages",
	"ast affected",
	"ast graph",
	"ast neighbors",
	"ast cut",

	"deport",
	"deport list",

	"gcal",
	"gcal auth",
	"gcal accounts list",
	"gcal setup check",
	"gcal config",
	"gcal list",
	"gcal export",
	"gcal delete",
	"gcal reschedule",
	"gcal create",
	"gcal preview",

	"gmail",
	"gmail auth",
	"gmail config",
	"gmail accounts",
	"gmail setup",
	"gmail spam clean",
	"gmail tidy",
	"gmail tidy audit",
	"gmail delete unread",
	"gmail timer",
	"gmail timer install",
	"gmail timer disable",
	"gmail timer status",
	"gmail contacts",
	"gmail preview",
	"gmail send",
	"gmail list",
	"gmail reply",
	"gmail open",
	"gmail inspect",
	"gmail mark",
	"gmail delete",
	"gmail spam",
	"gmail inspect-spam",

	"gdrive",
	"gdrive auth",
	"gdrive config",
	"gdrive sync run",
	"gdrive sync restore",
	"gdrive timer install",
	"gdrive timer disable",
	"gdrive timer status",
	"gdrive register",
	"gdrive list registrations",
	"gdrive remove registration",
	"gdrive restore registrations",
	"gdrive restore registration",

	"looptab",
	"looptab list",
	"looptab register",
	"looptab remove",
	"looptab show",
	"looptab edit",

	"pass",
	"pass create",
	"pass edit",
	"pass show",
	"pass delete",
	"pass upgrade",

	"slack",
	"slack accounts list",
	"slack setup check",
	"slack config",
	"slack auth",
	"slack mark",
	"slack contacts",
	"slack users",
	"slack events",
	"slack preview",
	"slack inspect",
	"slack send",
	"slack reply",
	"slack delete",
	"slack edit",
	"slack files",
	"slack open",
	"slack list",
	"slack unread",
	"slack thread",
	"slack conversations",

	"tunnel list",
	"tunnel add",
	"tunnel remove",
	"tunnel start",
	"tunnel stop",
	"tunnel status",
	"tunnel logs",
	"tunnel run",
}

var (
	known     map[string]struct{}
	astVerb   map[string]bool
	tuiScreen map[string]bool
)

func init() {
	known = make(map[string]struct{}, len(Catalog))
	astVerb = make(map[string]bool)
	tuiScreen = make(map[string]bool)
	for _, name := range Catalog {
		if _, dup := known[name]; dup {
			panic("duplicate usage feature " + name)
		}
		known[name] = struct{}{}
		if rest, ok := strings.CutPrefix(name, "ast "); ok && !strings.Contains(rest, " ") {
			astVerb[rest] = true
		}
		if rest, ok := strings.CutPrefix(name, "tui "); ok && !strings.Contains(rest, " ") {
			tuiScreen[rest] = true
		}
	}
}

// Known reports whether feature is in the catalog.
func Known(feature string) bool {
	_, ok := known[feature]
	return ok
}
