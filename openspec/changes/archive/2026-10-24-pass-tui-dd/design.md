## Context

`c3po tui gdrive`, `c3po tui deport`, `c3po tui o`, and `c3po tui xl` already delete with an armed `dd` chord. `c3po tui pass` lists vaults, opens one, and copies an item. The title-based delete is `c3po pass <vault> --delete <title>`, which runs `op item delete <id> --vault <vault>`.

## Goals / Non-Goals

**Goals:**

- Delete the selected item in the open vault with `dd`.
- Arm on the first `d`, show `d`, cancel with Esc, and let any other key run as itself.
- Reload the vault and keep the cursor at the same index.
- Leave a vault row in place.
- Keep the secret unread and unprinted.
- Name `dd` in the shortcut list and in `c3po tui pass help`.

**Non-Goals:**

- Deleting a vault.
- A separate y/n prompt.
- Changing `c3po pass <vault> --delete <title>`.
- Create, edit, show, and search on the screen.

## Decisions

- The chord matches Drive, deport, the navigator, and the spreadsheet. The first `d` sets the arm and the status `d`. The second `d` deletes. Esc clears the arm and the status and leaves the screen open. Any other key clears the arm and is handled as itself, including `q` and `j`. `?` clears the arm and opens the shortcut list. While that list is open, `d` does nothing.
- The other confirmation shape is a y/n prompt. The arm is the confirmation, matching the other screens.
- Delete calls `op item delete <id> --vault <vault name>` through a `Delete` method on the screen source. The vault name is the open vault. The id is the selected row. That is the same `op` invocation the title-based delete already uses after it resolves a title.
- A vault row sets the status `dd deletes an item` and makes no `op` call. An empty item list sets `no items`. An empty vault list sets `no vaults`.
- After a successful delete the vault reloads and the cursor is restored to the same index, then clamped. The status is `deleted` plus the item title. A failed delete leaves the listing in place and shows the failure.
- A missing vault name or item id fails before `op` runs. An `op` failure reports `Unable to delete the 1Password item.`

## Risks / Trade-offs

- [The second `d` removes the item] → The first `d` shows `d`, and Esc cancels the arm. The shortcut list names `dd`.
- [A service-account `op` needs `--vault`] → The call passes the open vault name, the same flag the title-based delete uses.
- [Reloading the vault resets the cursor] → The screen saves the index and restores it after the reload.

## Migration Plan

- No stored data changes. Install the updated `c3po` binary. The previous commit restores the screen that only copies.
- `c3po pass <vault> --delete <title>` continues to delete by exact title.

## Open Questions

- None.
