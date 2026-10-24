## Why

The spreadsheet can insert a row with `o` and `O`. Removing the current row still means clearing cells one at a time, and formulas keep pointing at the old positions.

## What Changes

- `dd` deletes the current row in `c3po tui xl`.
- The first `d` arms the chord and shows `d`. Esc cancels it. Any other key clears the arm and runs as a normal motion.
- Rows below move up. A reference to the deleted row becomes an error. A range that contained it shrinks.

## Impact

- Spec: `openspec/specs/xl/spec.md`
- Code: `xl/internal/sheet`, `xl/internal/tui`
