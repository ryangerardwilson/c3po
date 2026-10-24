package scout

import (
	"fmt"
	"io"

	"github.com/ryangerardwilson/c3po/ast"
)

const helpText = `c3po runs these apps.

Usage:
  c3po ast [args...]
  c3po deport list|<port>
  c3po gcal [args...]
  c3po gdrive [args...]
  c3po gmail [args...]
  c3po pass <vault> <item> [args...]
  c3po slack [args...]
  c3po tunnel add|remove|list|start|stop|status|logs|run
  c3po tui cyclotree|l|o|tunnel|gdrive|pass|deport|xl|looptab [args...]
  c3po version
  c3po help [-f <category> [feature]]

`

const helpFoot = `c3po help -f <category> explains that category.
c3po help -f ast <feature> explains one ast command.
`

const appText = `apps:
  ast       AST-aware edits, trace, and the agent boundary
  deport    free a safe external port
  gcal      Google Calendar across presets
  gdrive    local-first Google Drive sync
  gmail     Gmail send, list, spam clean, and the hourly timer
  pass      1Password vaults and items
  slack     Slack messages across presets
  tunnel    Cloudflare Tunnel routes for ryangerardwilson.com
  tui       terminal screens
`

const tuiText = `c3po tui runs a terminal screen. This is the only way to open one.

Usage:
  c3po tui cyclotree [symbol] [dir]
  c3po tui l [path] [flags]
  c3po tui o [args...]
  c3po tui tunnel
  c3po tui gdrive
  c3po tui pass
  c3po tui deport
  c3po tui xl [file]
  c3po tui looptab

screens:
  cyclotree    calls of one function, as a tree you move with hjkl
  l            compact colored tree of a directory
  o            vim-inspired file navigator
  tunnel       Cloudflare Tunnel route editor
  gdrive       Drive preset, then browse
  pass         1Password vaults, then copy one item
  deport       occupied external ports, freed with dd
  xl           spreadsheet for a budget workbook
  looptab      Looptab jobs, run one now

c3po help -f tui <screen> explains one screen.
`

func writeHelp(w io.Writer) {
	fmt.Fprint(w, helpText)
	fmt.Fprint(w, appText)
	fmt.Fprint(w, helpFoot)
}

func writeTUI(w io.Writer) {
	fmt.Fprint(w, tuiText)
}

func helpArgs(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeHelp(stdout)
		return 0
	}
	if args[0] != "-f" || len(args) > 3 {
		return helpUsage(stderr)
	}
	if len(args) == 1 {
		return helpUsage(stderr)
	}
	feature := ""
	if len(args) == 3 {
		feature = args[2]
	}
	return helpCategory(args[1], feature, stdout, stderr)
}

func helpUsage(stderr io.Writer) int {
	fmt.Fprintln(stderr, "usage: c3po help -f <category> [feature]")
	writeHelp(stderr)
	return 2
}

func helpCategory(cat, feature string, stdout, stderr io.Writer) int {
	if cat == "ast" {
		return helpAst(feature, stdout, stderr)
	}
	if cat == "tui" {
		return helpTUI(feature, stdout, stderr)
	}
	run, ok := apps[cat]
	if !ok {
		fmt.Fprintf(stderr, "unknown category %q\n", cat)
		writeHelp(stderr)
		return 2
	}
	return helpScout(feature, run, stdout, stderr)
}

func helpTUI(feature string, stdout, stderr io.Writer) int {
	if feature == "" {
		writeTUI(stdout)
		return 0
	}
	run, ok := screens[feature]
	if !ok {
		fmt.Fprintf(stderr, "unknown feature %q\n", feature)
		writeTUI(stderr)
		return 2
	}
	return run([]string{"help"}, stdout, stderr)
}

func helpScout(feature string, run runFunc, stdout, stderr io.Writer) int {
	if feature != "" {
		fmt.Fprintf(stderr, "unknown feature %q\n", feature)
		writeHelp(stderr)
		return 2
	}
	return run([]string{"help"}, stdout, stderr)
}

func helpAst(feature string, stdout, stderr io.Writer) int {
	if ast.Help(feature, stdout) {
		return 0
	}
	fmt.Fprintf(stderr, "unknown feature %q\n", feature)
	ast.Help("", stderr)
	return 2
}
