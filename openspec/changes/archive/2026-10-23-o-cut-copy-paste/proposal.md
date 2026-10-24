## Why

Moving or duplicating a file means leaving the navigator. The operator wants to stage the current file or directory, go to another directory, and put it there.

## What Changes

- `x` stages the current file or directory to move.
- `y` stages it to copy. It no longer sends the path to `wl-copy`.
- `p` places the staged item in the listed directory under the same name.
- A copy leaves the source in place and stays staged. A cut removes the source and clears the stage.
- Paste refuses when the name exists, when nothing is staged, or when a directory would be pasted into itself.

## Capabilities

### New Capabilities

### Modified Capabilities

- `o`: The navigator cuts, copies, and pastes the current file or directory.

## Impact

The navigator keys and the listed directory. No new dependency.
