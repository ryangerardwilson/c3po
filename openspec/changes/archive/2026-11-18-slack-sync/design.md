# Sync the slack desk

## Context

The desk stores an intake time and prints it as the watching-after clock. That time is how far back a refresh searches. It is not the age of the rows on the screen. The board's `refreshed_at` is when the desk last finished a refresh, stored as UTC RFC3339. `rgw-slack-quickshell-plugin refresh` searches Slack and classifies. The screen already reloads the board with `show`.

## Goals

- Remove the intake line.
- `s` refreshes the desk from Slack and reloads the table.
- Show the last refresh as `last synced at` in India time when the stored value is RFC3339.

## Non-goals

- Changing the desk's intake time or its refresh rules.
- Searching Slack from the screen itself.
- Posting a message.
- A second scheduler. The desk timer still refreshes on its own.

## Decisions

- `s` runs `rgw-slack-quickshell-plugin refresh`, then `show`. The screen paints `syncing` before the refresh returns. A command failure leaves the current rows and shows the failure. `refresh already running` is a failure even when the command exits 0.
- `last synced at` uses the board's `refreshed_at`. RFC3339 values are shown in Asia/Kolkata as `Mon, 2 Jan 2006, 3:04 PM MST`. Any other value is shown as stored.
- An empty workspace says `no items`. The intake time stays in the desk and is not painted.
- While the shortcut list is open, `s` does not refresh.

## Risks

- A refresh can take a few minutes. The screen waits and shows `syncing` until it finishes. Keys are not read during that wait.
- A refresh updates both workspaces. The open workspace and row stay selected when they are still on the desk.
