# Sync the slack desk

## Why

The slack screen says `watching after` and an intake time. That clock is the desk's search floor, so it reads as if the rows start then. The screen also has no way to fetch what Slack has now. The refresh time is an unlabeled timestamp.

## What Changes

- The screen no longer says `watching after`.
- An empty workspace says `no items`.
- `s` asks the desk to refresh from Slack and reloads the table. The status says `syncing` while that runs. A failed refresh leaves the rows in place.
- The screen says `last synced at` and the desk's refresh time.

## Capabilities

### New Capabilities

### Modified Capabilities

- `slack`: Drop the intake line. `s` syncs from Slack and the screen shows when the desk last synced.
- `tui`: The slack shortcut list names sync.

## Impact

- Specs: `openspec/specs/slack/spec.md`, `openspec/specs/tui/spec.md`
- Code: `slack/internal/tui`, `slack/screen.go`
- Sync runs `rgw-slack-quickshell-plugin refresh`, the same refresh Super+S uses. The screen does not post. The Quickshell desk is unchanged.
