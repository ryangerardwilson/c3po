# Slack desk on the r2d2 screen

## Why

`r2d2 slack` lists workspaces, then conversations, then messages. Super+S shows a different table: direct messages to answer, messages that tag Ryan, and messages he should know, each with the time, the person, and the text. The screen does not show that table.

## What Changes

- `r2d2 slack` shows the Slack desk for the current workspace.
- The table has the sections `DMS`, `TAGGED`, and `SHOULD KNOW`, with a count on each.
- A row shows the time, the sender or the conversation, and the desk reason when one is set, otherwise the message text.
- `h` and `l` switch workspace and wrap. Enter opens the row through the desk and does not mark it read. `r` marks the row read on the desk and drops it.
- The screen does not search Slack and does not post.

## Capabilities

### New Capabilities

### Modified Capabilities

- `slack`: The screen is the desk table. Opening a row does not mark it read.
- `tui`: The slack shortcut list names the desk. The screen has no compose line.

## Impact

- Specs: `openspec/specs/slack/spec.md`, `openspec/specs/tui/spec.md`
- Code: `slack/internal/tui`, `slack/screen.go`, the screen help tests, and the c3po readme
- The Quickshell desk still searches Slack and classifies. The screen reads that desk with `rgw-slack-quickshell-plugin show`, opens with `open`, and marks with `read`.
- `c3po slack` is unchanged. The session still does not store the user token. A key still does not publish the backup.
