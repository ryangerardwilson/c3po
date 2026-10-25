## Why

The navigator can cut, copy, or delete only the current row. Moving or removing a run of files means repeating that one row at a time.

## What Changes

- `v` toggles a visual range on consecutive listing rows. `j`, `k`, and the arrow keys extend it. The range does not wrap.
- Delete removes every file and directory in the range. `dd` does that too while visual mode is on, and still deletes only the current row otherwise.
- `x` stages the range to move. `y` stages it to copy. `p` pastes every staged item into the listed directory.
- A paste that would collide with an existing name, or put a directory inside itself, changes nothing.

## Capabilities

### New Capabilities

### Modified Capabilities

- `o`: The navigator selects a consecutive range and cuts, copies, pastes, or deletes that range.

## Impact

The navigator keys and the clipboard stage. No new dependency.
