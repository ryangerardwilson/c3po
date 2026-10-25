## Why

`touch` creates a file and leaves existing text in place. The colon line has no way to create or empty a file by redirection.

## What Changes

- `!> hello.txt` creates `hello.txt` or empties it.
- That redirect stays inside the navigator and selects the file.
- `touch` still leaves existing text in place.
- Any other `!` line still runs in the shell.

## Capabilities

### New Capabilities

### Modified Capabilities

- `o`: A bang redirect of one path creates or empties that file.

## Impact

The navigator colon line on `c3po tui o`. `mv`, `rm`, `cp`, `touch`, and `mkdir` stay as they are. A `!` line that is not `>` and one path still hands the terminal to the shell.
