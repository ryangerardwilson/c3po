## Why

The spreadsheet lives in a separate program, `xl`. The file navigator opens an `.xlsx` file with the desktop opener, so a budget workbook leaves the terminal.

## What Changes

- `c3po tui xl` is the spreadsheet screen. The `xl` command name runs that same screen.
- `c3po tui o` opens a file whose name ends in `.xlsx` in that screen, in the foreground. Quitting the spreadsheet returns to the navigator. Other non-text files still go to xdg-open.
- The spreadsheet source moves into this repository. The standalone tree is retired.

## Capabilities

### New Capabilities

- `xl`: The spreadsheet screen, reached as `c3po tui xl` or as the `xl` command.

### Modified Capabilities

- `scout`: The terminal screens include `xl`.
- `o`: An `.xlsx` path opens in the spreadsheet screen.

## Impact

`c3po tui` dispatch, the `o` opener, and the installer. The module gains the spreadsheet's dependencies. `c3po xl` stays an unknown command. A text file still opens in the editor.
