## Why

`c3po tui pass` can open a vault and copy an item. Removing an item still requires leaving the screen for `c3po pass <vault> --delete <title>`.

## What Changes

- `dd` deletes the selected item in the open vault.
- The first `d` arms the chord and shows `d`. Esc cancels it and leaves the screen open. Any other key clears the arm and runs as that key.
- A vault row stays. An empty item list reports `no items`. An empty vault list reports `no vaults`.
- The shortcut list and `c3po tui pass help` name `dd`. The secret stays unread and unprinted.
- `c3po pass <vault> --delete <title>` stays the title-based delete.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `tui`: `dd` deletes the selected pass item, and the shortcut list names `dd`.
- `scout`: `c3po tui pass help` says `dd` deletes the selected item.

## Impact

- Code: `pass/internal/tui`, `pass/internal/op`, and the pass screen help.
- Specs: `openspec/specs/tui/spec.md` and `openspec/specs/scout/spec.md`.
- No new dependency, data migration, or version bump.
