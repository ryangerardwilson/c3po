package ast

import (
	"io"

	"github.com/ryangerardwilson/c3po/ast/internal/cli"
)

// Run executes ast.
func Run(args []string, stdout, stderr io.Writer) int {
	runner := cli.NewRunner(stdout, stderr)
	return runner.Run(args)
}

// Help writes one ast command, or the ast command list when feature is empty.
// It reports whether that feature exists.
func Help(feature string, w io.Writer) bool {
	text, ok := cli.HelpText(feature)
	if !ok {
		return false
	}
	_, _ = io.WriteString(w, text)
	return true
}
