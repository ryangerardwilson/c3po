package ast

import (
	"io"

	"github.com/ryangerardwilson/c3po/ast/internal/cli"
)

// Run executes the ast scout.
func Run(args []string, stdout, stderr io.Writer) int {
	runner := cli.NewRunner(stdout, stderr)
	return runner.Run(args)
}
