## Why

The slack screen shows the desk that was last refreshed. New Slack messages stay off the table until the operator presses `s`. The desk's own timer is 30 minutes, so an open screen goes stale.

## What Changes

- While `r2d2 slack` stays open, the screen asks the desk to refresh about every 10 minutes, on the same path as `s`.
- The status says `syncing` before that refresh returns. A failed refresh leaves the rows, shows the failure, and waits about 10 minutes before the next one.
- The screen does not refresh when it opens. It does not refresh while the shortcut list is open. After that list closes, a refresh that came due during the list runs.
- Pressing `s` still refreshes and starts the 10 minute wait again.
- Help says the screen syncs about every 10 minutes.

## Capabilities

### New Capabilities

### Modified Capabilities

- `slack`: An open screen refreshes the desk about every 10 minutes on the same path as `s`.

## Impact

- Spec: `openspec/specs/slack/spec.md`
- Code: `slack/internal/tui`, `slack/screen.go`, and the screen sentences in the README and AGENTS.md
- The refresh is `rgw-slack-quickshell-plugin refresh`, then `show`. The screen does not post and does not mark a row read.
- The desk's 30 minute timer stays 30 minutes. The screen's 10 minute place publish stays a separate timer.
