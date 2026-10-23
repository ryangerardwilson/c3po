package cli

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

const helpText = `xl

A grid for a budget. Type expenses and income, keep formulas, save an xlsx.

global actions:
  xl help
    show this help
  xl version
    print the installed version

features:
  open a blank sheet
  # open
  c3po tui xl
  xl
  xl open

  create or open an xlsx
  # <file>
  c3po tui xl budget
  c3po tui xl budget.xlsx
  xl budget
  xl budget.xlsx
  xl open budget.xlsx

  work on more than one sheet
  # sheets
  [ and ] switch sheets
  n adds a sheet
  R renames the current sheet
  =Income!B1
`

func WriteHelp(out io.Writer) {
	if file, ok := out.(*os.File); ok && term.IsTerminal(int(file.Fd())) && os.Getenv("NO_COLOR") == "" {
		_, _ = fmt.Fprintf(out, "\033[38;5;245m%s\033[0m", helpText)
		return
	}
	_, _ = io.WriteString(out, helpText)
}
