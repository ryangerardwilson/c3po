# Slack desk on the r2d2 screen

## Context

Super+S reads `rgw-slack-quickshell-plugin show`. The board has one workspace list. Each workspace has `dms`, `tagged`, and `aware`. A row has a time, a sender, a conversation, text, and sometimes a reason. The overlay prints three heads, `DMS`, `TAGGED`, and `SHOULD KNOW`, then one line per row: time, person, and the reason or the text. `h` and `l` wrap across workspaces. Enter runs `open` and leaves the overlay. `r` runs `read`. Opening does not mark the row read.

`r2d2 slack` instead lists the account file and calls Slack for conversations and history.

## Goals

- The screen shows the same three sections and the same fields as the desk.
- Enter opens the desk row and does not mark it read. `r` marks it read on the desk.
- The screen does not search Slack and does not post.

## Non-goals

- Replacing the Quickshell overlay.
- Classifying messages inside the screen.
- A compose line, a reply, or `conversations.mark`.
- Painting any color other than black, white, or gray.

## Decisions

- The screen loads the board with `rgw-slack-quickshell-plugin show`, opens a row with `open`, and marks a row with `read`. Those commands already apply the desk's read state, clocks, and browser focus. The screen does not read the desk sqlite itself.
- A failed `show` opens the screen with that failure and lists no rows. A failed `open` or `read` leaves the row in place.
- A successful open leaves the screen so the browser can take focus, which is what the overlay does.
- The table is `WHEN`, `WHO`, and `WHAT`. `WHO` is the sender, or the conversation when the sender is empty. `WHAT` is the reason when it is set, otherwise the text. Whitespace in a cell becomes a single space.
- Section heads stay visible when a section is empty. A workspace with no rows says `watching after` and the intake time, or `no items`. No workspace and no refresh time says `no brief yet`.
- The stored place is the workspace preset and the row's message id. It does not contain the token. A key still does not publish.

## Risks

- The table is whatever the desk last refreshed. The screen does not start a refresh.
- A successful open leaves the screen. The operator opens `r2d2 slack` again to return to the table.
- `show` can write the desk board when the desk's focus state is first created. That is the same write the overlay performs on open.
