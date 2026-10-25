## Context

`x`, `y`, and `p` stage one path. `dd` deletes the current row. Marks (`m`) pick scattered rows for another app. There is no way to act on a consecutive run.

## Goals / Non-Goals

**Goals:**

- `v` selects the inclusive run from an anchor row to the cursor.
- Delete, `x`, and `y` act on that run, then leave visual mode.
- `p` places every staged path in the listed directory, in staged order.

**Non-Goals:**

- Selecting rows that are not next to each other. `m` still does that for pick.
- Visual mode on any screen but the navigator.
- Pasting over a name that already exists.

## Decisions

The range is two indexes into the current listing. Motion keys move the cursor and stop at the ends while visual mode is on, so the run cannot wrap around the list. `v` again, or Esc, clears the range without touching files. A listing rebuild clears it too, so a reload cannot leave the anchor pointing at a different row.

The stage is an ordered list of paths. Outside visual mode, `x` and `y` still stage the current row. Inside it, they stage every path from the top of the range to the bottom, then leave visual mode. Paste checks every destination before it writes. If any name exists, any source is missing, or any directory would be pasted into itself, nothing is written and the stage stays. A copy stays staged. A cut clears the stage only after every item lands.

Delete and `dd` share one removal. The Delete key removes the range and does nothing when visual mode is off. `dd` removes the range while visual mode is on and the current row otherwise. A staged path that was removed, or that lived inside a removed directory, leaves the stage. Paths that were not removed stay staged.

The range rows use the same highlight as the cursor. The cursor row keeps the `>` marker.

## Risks / Trade-offs

A filesystem change that rebuilds the listing drops the range. The operator presses `v` again.

A refused multi-file paste leaves every source where it was, including when only one of the names collides.

## Migration Plan

Single-row `x`, `y`, `p`, and `dd` stay. `y` still does not send the path to `wl-copy`.

## Open Questions

None.
