## Why

`!` opens the touch, mkdir, and rm line, so it cannot run a shell command. Backspace does not delete a visual range.

## What Changes

- **BREAKING** `:` opens the bottom command line for `touch`, `mkdir`, and `rm`. `!` no longer does.
- `!` opens a bottom command line that runs the typed line in the shell, with the listed directory as its working directory. The terminal is handed to that command, and the listing reloads when it exits.
- Backspace deletes the visual range. Outside visual mode it does not delete. In the filter or a command line it still edits text.

## Capabilities

### New Capabilities

### Modified Capabilities

- `o`: The navigator opens file commands with `:`, runs a shell command with `!`, and deletes a visual range with Backspace.

## Impact

Navigator keys, the command runner, and help. No new dependency.
