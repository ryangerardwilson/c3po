## Why

The bang is a separate prompt, and the colon line has a space after `:`. The command line only appends at the end, so Emacs editing keys do nothing.

## What Changes

- `!` and a colon line that starts with `!` show `:!` with the bang against the colon, and run the rest in the shell.
- Other colon lines still run only mv, rm, cp, touch, and mkdir, with no hint. `rm` may take `-rf`.
- The command line has a cursor. Ctrl-a, Ctrl-e, Ctrl-w, Alt-b, Alt-f, and the other readline Emacs keys edit that line.

## Capabilities

### New Capabilities

### Modified Capabilities

- `o`: The colon line shows `:!` and edits like a readline Emacs line.

## Impact

The navigator command line, its help, and the shell path for a leading bang. File commands that do not start with `!` stay on the allowlist.
