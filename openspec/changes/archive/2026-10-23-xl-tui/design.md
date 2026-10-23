## Context

The spreadsheet is the Go module `github.com/ryangerardwilson/xl` at `Apps/xl` in the home repo. Its command parses `xl`, `xl <file>`, `xl open`, `help`, and `version`, then runs a Bubble Tea grid. c3po screens are names under `c3po tui`. `o` opens a text file in `$VISUAL`, `$EDITOR`, or vim and hands every other file to xdg-open. An `.xlsx` file is not text-like, so the navigator detaches xdg-open. c3po does not depend on Bubble Tea.

## Goals / Non-Goals

**Goals:**

- One spreadsheet, inside c3po, launched as `c3po tui xl` and as `xl`.
- Opening an `.xlsx` file from `o` runs that screen in the foreground and returns to the navigator.
- Keep the grid, the formulas, and the keys the spreadsheet already has.

**Non-Goals:**

- A top-level `c3po xl` app.
- Opening `.csv` in the spreadsheet. A csv still follows the non-text path.
- Publishing a new version tag.
- Restating the formula language in this spec. The moved tests keep that contract.

## Decisions

The packages move to `xl/` in this module. Imports change from `github.com/ryangerardwilson/xl` to `github.com/ryangerardwilson/c3po/xl`. `xl.Run` is the existing command runner, so `c3po tui xl budget.xlsx` and `xl budget.xlsx` share one path. The screen accepts a path. It does not reject extra arguments the way `deport` does.

The installer links `xl` to the c3po binary, the same way it links `rgw-ast`. The program name `xl` selects the spreadsheet. `c3po xl` stays unknown because a screen is only `c3po tui <name>`.

`o` recognizes a file by the `.xlsx` suffix, ignoring case. It starts the current executable with the arguments `tui`, `xl`, and the path, and waits. The navigator uses the same foreground handoff it uses for the editor, so the spreadsheet takes the terminal and quitting it paints the navigator again. A direct `c3po tui o file.xlsx` uses that same command and then exits. A set of only text files still opens together in the editor. Any other non-xlsx file is still detached.

The spreadsheet version string stays `0.1.0`. `c3po version` stays the c3po version.

## Risks / Trade-offs

- [Bubble Tea joins this module] → The dependency is the grid the screen already runs. The other screens stay hand-rolled.
- [A second c3po process holds the terminal while a workbook is open] → That is the same handoff as vim. The navigator does not nest two raw terminals.
- [The home `xl` binary is replaced by the link] → `xl budget.xlsx` still opens the workbook. Quit a spreadsheet that was already running, then start it again.
