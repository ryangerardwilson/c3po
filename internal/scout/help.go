package scout

import (
	"fmt"
	"io"

	"github.com/ryangerardwilson/c3po/ast"
)

const helpText = `c3po runs these apps.

Usage:
  c3po ast [args...]
  c3po cyclotree [symbol] [dir]
  c3po deport <port>
  c3po gcal [args...]
  c3po gdrive [args...]
  c3po gmail [args...]
  c3po l [args...]
  c3po o [args...]
  c3po pass <vault> <item> [args...]
  c3po tunnel open|add|remove|list|start|stop|status|logs|run
  c3po version
  c3po help [-f <category> [feature]]

`

const helpFoot = `c3po help -f <category> explains that category.
c3po help -f ast <feature> explains one ast command.
`

const appText = `apps:
  ast          AST-aware edits, trace, and the agent boundary
  cyclotree    calls of one function, as a tree you move with hjkl
  deport       free a port by signalling its TCP and UDP holders
  gcal         Google Calendar across presets
  gdrive       local-first Google Drive sync
  gmail        Gmail send, list, spam clean, and the hourly timer
  l            compact colored tree of a directory
  o            vim-inspired file navigator
  pass         1Password vaults and items
  tunnel       Cloudflare Tunnel routes for ryangerardwilson.com
`

func writeHelp(w io.Writer) {
	fmt.Fprint(w, helpText)
	fmt.Fprint(w, appText)
	fmt.Fprint(w, helpFoot)
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
	run, ok := apps[cat]
	if !ok {
		fmt.Fprintf(stderr, "unknown category %q\n", cat)
		writeHelp(stderr)
		return 2
	}
	return helpScout(feature, run, stdout, stderr)
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

