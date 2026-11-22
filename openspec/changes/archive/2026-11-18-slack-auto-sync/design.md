## Context

`s` paints `syncing`, runs `rgw-slack-quickshell-plugin refresh`, then reloads the board with `show`. The key loop blocks in one stdin read, so a timer cannot fire during that read unless the wait itself has a deadline. The place snapshot timer publishes the session every 10 minutes and does not refresh Slack. A user systemd timer refreshes the desk every 30 minutes whether or not the screen is open.

## Goals / Non-Goals

**Goals:**

- An open screen refreshes about every 10 minutes on the same path as `s`.
- The first refresh waits about 10 minutes after the screen opens.
- Manual `s` starts that wait again.
- The shortcut list delays the refresh until the list closes, and the paint loop does not spin.
- A failed refresh leaves the rows and waits about 10 minutes.

**Non-Goals:**

- Changing the 30 minute desk timer.
- A second classifier or a refresh goroutine.
- Publishing the session when the automatic refresh finishes.
- Refreshing Slack from the screen by any path other than the desk command.
- Syncing when the screen opens.

## Decisions

- `syncEvery` is 10 minutes and is separate from `snapshotEvery`. The loop records the next refresh time. `waitKey` uses `readUntil` when the terminal implements it. A test screen that only has `readKey` never times out, so existing key tests stay on the key path.
- The real terminal polls stdin until the deadline. One read happens only after the poll says the terminal is readable. There is no background reader, so leaving the screen does not steal the next key.
- While the shortcut list is open the deadline is 24 hours out. The interval is not polled, so the paint loop does not spin. After the list closes, a deadline that has already passed returns at once and the refresh runs.
- Automatic refresh calls the same `sync` as `s`: paint `syncing` first, leave the rows on failure, and treat `refresh already running` as a failure. The next wait is 10 minutes after that call returns, including after a failure, so a failure does not retry immediately.
- Keys are not read during the refresh. A second refresh cannot start until this one returns.

## Risks / Trade-offs

- A refresh can take a few minutes. The screen shows `syncing` and does not read keys until it finishes. That matches `s`.
- The 30 minute timer can overlap an open screen. `refresh already running` stays a failure, the rows stay, and the screen tries again about 10 minutes later.
- While the screen is open the desk refreshes more often than every 30 minutes. The closed-screen timer stays 30 minutes.

## Migration Plan

- No stored data changes. An already-open slack screen keeps its current binary until it quits.
