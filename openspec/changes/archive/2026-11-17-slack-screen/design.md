# Slack screen

## Context

`c3po slack` already holds the user tokens and the Slack Web API client. r2d2 screens other than `web` run through `c3po/screen`. The Quickshell desk is a separate app and is not this screen.

## Goals

- One screen, `r2d2 slack`, for reading a workspace and answering in it.
- Same account file as `c3po slack`. No second token store.
- Vim motion and the hidden shortcut list used by the other screens.

## Non-goals

- Replacing `c3po slack`.
- Classifying mail with a model, refreshing a desk, or focusing a browser.
- Changing the Quickshell plugin.

## Decisions

- The screen starts on the workspace list. `h` and `l` move between presets there and on the conversation list.
- History is oldest at the top. The cursor starts on the newest message.
- `i` posts to the channel. `r` posts with `thread_ts` set to the selected message's thread, or to that message when it is the root.
- `m` calls `conversations.mark`. Loading history does not.
- The session path is the preset id. The cursor is the row key. The token is never part of the model or the session.

## Risks

- A workspace with many conversations waits on `users.conversations`. The screen shows the error and stays open.
- A direct message whose Slack object has no display name is labeled with the user id until `users.info` resolves it.

## Migration

No account migration. An operator who already ran `c3po slack auth` opens `r2d2 slack`.
