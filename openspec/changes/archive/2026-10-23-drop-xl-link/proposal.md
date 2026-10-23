## Why

The spreadsheet is `c3po tui xl`. A separate `xl` command on `PATH` is the old program name, and the installer still links it to the c3po binary.

## What Changes

- The installer does not link `xl`. An `xl` symlink that points at c3po is removed.
- Invoking the binary as `xl` does not open the spreadsheet.
- `c3po tui xl` is unchanged. `o` still opens an `.xlsx` file in that screen.

## Capabilities

### New Capabilities

### Modified Capabilities

- `scout`: The program name `xl` no longer selects the spreadsheet.

## Impact

The installer and the program-name dispatch. No change to the grid.
