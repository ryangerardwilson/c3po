## Why

The colon line prints a hint and refuses mv, cp, and `rm -rf`. A line that starts with `!` also runs an arbitrary shell command.

## What Changes

- The colon line shows `:` and the text being typed. It does not print a hint.
- It runs only `mv`, `rm`, `cp`, `touch`, and `mkdir`. `rm` may take `-rf`. Any other command or flag is refused. A leading `!` is not a shell.

## Capabilities

### New Capabilities

### Modified Capabilities

- `o`: The colon command line runs mv, rm, cp, touch, and mkdir, and shows no hint.

## Impact

The navigator colon command line, its allowlist, and its help. The `!` key is unchanged.
