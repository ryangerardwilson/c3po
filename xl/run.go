// Package xl is the r2d2 xl spreadsheet.
package xl

import (
	"io"

	"github.com/ryangerardwilson/c3po/xl/internal/cli"
)

// Run executes the spreadsheet. Arguments are the ones after "xl".
func Run(args []string, stdout, stderr io.Writer) int {
	return cli.NewRunner(stdout, stderr).Run(args)
}
