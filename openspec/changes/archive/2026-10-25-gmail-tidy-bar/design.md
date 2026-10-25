## Context

Cleanup prints `processed group i/n` for every batch of senders. Tidy calls that for each preset.

## Goals / Non-Goals

- Goals: one line per preset, a block bar, in-place updates on a terminal, a finished line that names the trashed count.
- Non-Goals: changing which messages are trashed, or the audit list.

## Decisions

- The line is `preset  email  bar  done/total`, and the finished line adds `trashed N`.
- A terminal redraws with carriage return and a clear-to-end. Color is used only when the terminal is live, `NO_COLOR` is unset, and `TERM` is not `dumb`.
- Any other stdout gets the finished line once. The per-group lines go away.
- Tidy does not print the `preset= email=` banner. The progress line carries that.
- `c3po gmail spam clean` uses the same bar. It keeps the `preset= email=` banner only when stdout is not a terminal.

## Risks / Trade-offs

- A narrow terminal can wrap the line. The bar stays 16 cells so the line stays short.

## Migration Plan

Run `c3po gmail tidy`. Each account is one bar. The last line is still the completion summary.

## Open Questions

None.
