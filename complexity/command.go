package complexity

import (
	"io"

	"github.com/ryangerardwilson/c3po/complexity/internal/cli"
)

// Run executes the complexity scout.
func Run(args []string, stdout, stderr io.Writer) int {
	return cli.Execute(args, stdout, stderr)
}
