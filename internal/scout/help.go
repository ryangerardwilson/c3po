package scout

import (
	"fmt"
	"io"

	"github.com/ryangerardwilson/c3po/ast"
	"github.com/ryangerardwilson/c3po/cyclotree"
)

const helpText = `c3po runs scouting tools.

Usage:
  c3po scout ast [args...]
  c3po scout cyclotree [symbol] [dir]
  c3po version
  c3po help [-f <category> [feature]]

`

const helpFoot = `c3po help -f <category> explains that category.
c3po help -f ast <feature> explains one ast command.
`

const scoutText = `scouts:
  ast          AST-aware edits, trace, and the agent boundary
  cyclotree    calls of one function, as a tree you move with hjkl
`

func writeHelp(w io.Writer) {
	fmt.Fprint(w, helpText)
	fmt.Fprint(w, scoutText)
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
	switch cat {
	case "ast":
		return helpAst(feature, stdout, stderr)
	case "cyclotree":
		return helpScout(feature, cyclotree.Run, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown category %q\n", cat)
		writeHelp(stderr)
		return 2
	}
}

func helpScout(feature string, run func([]string, io.Writer, io.Writer) int, stdout, stderr io.Writer) int {
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

func writeScouts(w io.Writer) {
	fmt.Fprint(w, scoutText)
}
