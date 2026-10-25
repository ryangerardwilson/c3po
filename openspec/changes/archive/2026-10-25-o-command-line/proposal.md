## Why

`!` asks for `touch`, `mkdir`, or `rm` on the prompt line under the title. The operator types those commands at the top of the screen, above the listing.

## What Changes

- `!` opens a command line on the last row.
- The typed command is shown there, after `!`, and is not shown under the title.
- Enter and Esc keep their current behavior. The allowed commands stay `touch`, `mkdir`, and `rm`.

## Capabilities

### New Capabilities

### Modified Capabilities

- `o`: The `!` command line is the last row while it is open.

## Impact

The navigator view. No new dependency. A command other than `touch`, `mkdir`, or `rm` still does not run.
