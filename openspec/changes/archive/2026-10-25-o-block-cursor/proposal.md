## Why

The command line draws a vertical bar. That is a beam, and the rest of the navigator does not ask the terminal for a block cursor.

## What Changes

- The command line and the filter draw the cursor as one reverse-video cell.
- The cell covers the character under the cursor, or a space at the end of the text.
- While the navigator owns the terminal, the terminal cursor is a steady block. It returns to the terminal default when the navigator lets the terminal go.

## Capabilities

### New Capabilities

### Modified Capabilities

- `o`: The navigator draws a block cursor instead of a beam.

## Impact

The navigator command line, the filter line, and the terminal cursor shape for `c3po tui o`. The listing marker stays `>`.
