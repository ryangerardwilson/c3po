## Why

`j` and `k` stop at the ends of the lists on the r2d2 screens that live in c3po. Moving past the last row should return to the first, and moving above the first should return to the last.

## What Changes

- On slack, deport, looptab, pass, the Drive file list, the Drive preset list, and cyclotree, `j` and `k` wrap the list.
- An empty list stays put. A list of one row stays on that row.
- `g` and `G` still jump to the first and last row where those keys exist.
- While `?` is open, `j` and `k` still do nothing.
- The visual range in `o` still does not wrap. The spreadsheet still stops at the sheet edges.
- Deleting a row still leaves the cursor at the same index, clamped to the remaining rows.

## Capabilities

### New Capabilities

### Modified Capabilities

- `slack`: `j` and `k` wrap the desk rows.
- `deport`: `j` and `k` wrap the port list.
- `looptab`: `j` and `k` wrap the job list.
- `cyclotree`: `j` and `k` wrap the visible rows and stay inside the tree.
- `tui`: `j` and `k` wrap the Drive file list, the Drive preset list, and the pass vault and item lists.

## Impact

- Specs: slack, deport, looptab, cyclotree, and tui.
- Code: the `j` and `k` path on those screens.
- The screen menu and the web lists are specified in r2d2. The tunnel route list is specified in nebula.
- The spreadsheet, the `o` visual range, slack workspace `h` and `l`, and the delete clamp stay as they are.
