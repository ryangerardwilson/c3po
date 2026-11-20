## Context

Each screen owns its cursor. Several lists increment the cursor and then clamp it to the ends. `o` already wraps when visual mode is off and clamps when visual mode is on. The spreadsheet clamps to the sheet. Slack `h` and `l` already wrap the workspaces.

## Goals / Non-Goals

**Goals:**

- `j` and `k`, and the paired down and up keys, wrap every r2d2 list cursor in this repo except the spreadsheet and a visual range.
- An empty list does not move. One row stays on that row.
- Help still swallows `j` and `k`.
- `g` and `G` still jump to the ends.
- A delete still clamps the cursor to the remaining rows.

**Non-Goals:**

- Wrapping the spreadsheet grid.
- Wrapping the `o` visual range.
- Changing slack workspace `h` and `l`.
- A shared cursor package.
- The r2d2 menu, the web lists, and the nebula tunnel route list. Those are specified in their own repositories.

## Decisions

- Wrap with `(cursor + delta%n + n) % n` when the list has rows. Go's remainder keeps the sign of the dividend, so the added `n` keeps the index non-negative.
- The existing clamp used after a delete stays a clamp. Only the `j` and `k` path wraps. The clamp that runs after that path is then a no-op when the index is already in range.
- Cyclotree wraps among the visible rows, so the cursor stays inside the tree. Opening the first child of a row is a separate step and does not wrap.
- Each package keeps its own small helper. The screens do not share a module.

## Risks / Trade-offs

- `k` on the first cyclotree row now lands on the last visible row. That used to stay put.
- A wrap test has to press `j` or `k` past the end. Tests that stop one row early stay valid.

## Migration Plan

- No stored data changes. An already-open screen keeps its current binary until it quits.
