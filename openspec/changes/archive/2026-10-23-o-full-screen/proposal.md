## Why

The navigator paints only as many lines as the listing has, so a short directory sits in a band and leaves the rest of the terminal unused. Creating or removing a file means leaving the screen.

## What Changes

- The navigator view is exactly as tall as the terminal. The title is the first line and the key hint is the last line. A short listing pads the rows between them.
- `!` opens a prompt. Enter runs `touch`, `mkdir`, or `rm` in the listed directory. `mkdir` may take `-p`. Any other command or flag is refused. Esc cancels the prompt. The listing reloads after a command runs.

## Capabilities

### New Capabilities

### Modified Capabilities

- `o`: The screen fills the terminal height and runs touch, mkdir, and rm from `!`.

## Impact

The navigator view, its key help, and the listed directory. No new dependency. A command other than touch, mkdir, or rm does not run.
