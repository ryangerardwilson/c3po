## Context

`y` sends marked paths, or the current path, to `wl-copy`. There is no way to move or duplicate a file inside the navigator.

## Goals / Non-Goals

**Goals:**

- `x` cuts the current row, `y` copies it, and `p` pastes it into the listed directory.
- Directories move and copy with their contents.

**Non-Goals:**

- Pasting several marked rows.
- Sending paths to the desktop clipboard.
- Overwriting a name that already exists.

## Decisions

The stage is one path on the model, so it survives navigation. Paste uses the listed directory and the source base name. Copy walks the tree without following directory symlinks. Move uses rename, and copies then removes the source when the destination is another filesystem. A cut clears the stage after a successful paste. A copy stays staged.

## Risks / Trade-offs

A name collision is refused. The operator renames or removes the existing item first.

## Migration Plan

`y` stages a file copy. The clipboard command is gone.

## Open Questions

None.
