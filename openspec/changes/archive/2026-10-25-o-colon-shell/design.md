## Context

The navigator already draws a command line on the last row. It accepts only `touch`, `mkdir`, and `rm`, and `!` opens it. Visual mode deletes with the Delete key and with `dd`.

## Goals / Non-Goals

- Goals: `:` runs the same file commands. `!` runs any shell command in the listed directory. Backspace deletes the visual range.
- Non-Goals: changing which flags `touch`, `mkdir`, and `rm` accept. A `cd` inside the shell does not move the navigator.

## Decisions

- `commanding` means a bottom line is open. `shelling` chooses `!` over `:`.
- File commands stay in `applyFileCommand`. A shell line is `$SHELL -c`, or `/bin/sh -c` when `SHELL` is empty, with `Dir` set to the listed directory. `tui.Exec` hands over the terminal. An empty shell line does not run.
- After the shell exits, the listing reloads. A non-zero exit sets the status to the line plus ` failed`.
- Backspace and `ctrl+h` share the Delete path. That path stays behind the filter, the command line, and the help overlay, so those modes still edit text or ignore the key.
- The shortcut list stays 24 lines by dropping the blank line between the intro and the keys.

## Risks / Trade-offs

- `!` now runs whatever the operator types. That is the requested shell. The working directory is the directory on screen.
