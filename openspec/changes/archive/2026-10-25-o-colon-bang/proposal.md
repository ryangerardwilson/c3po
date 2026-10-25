## Why

The colon command line refuses every command except touch, mkdir, and rm. Vim runs any shell command from that line when it starts with `!`.

## What Changes

- A colon line that starts with `!` runs the rest in the shell in the listed directory. The terminal is handed to that command. An empty command does not run.
- A colon line that does not start with `!` still accepts only touch, mkdir, and rm.

## Capabilities

### New Capabilities

### Modified Capabilities

- `o`: The colon command line runs a shell command when the line starts with `!`.

## Impact

The navigator colon command line and its help. No new dependency.
