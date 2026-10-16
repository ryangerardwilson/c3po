package scout

import (
	"fmt"
	"io"
)

const helpText = `c3po runs scouting tools.

Usage:
  c3po scout ast [args...]
  c3po scout vibetree <dir> [-p N] [-m loc|cyclo] [--json]
  c3po version
  c3po help

Run c3po scout for the tools.
`

const scoutText = `scouts:
  ast          AST-aware edits, trace, and the agent boundary
  vibetree     cyclomatic or line counts by percentile, as a tree of files and directories
`

func writeHelp(w io.Writer) {
	fmt.Fprint(w, helpText)
}

func writeScouts(w io.Writer) {
	fmt.Fprint(w, scoutText)
}
