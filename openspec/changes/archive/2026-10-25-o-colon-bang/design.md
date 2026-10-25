## Context

`:` opens a command line limited to touch, mkdir, and rm. `!` already opens a separate line that runs any shell command in the listed directory.

## Goals / Non-Goals

- Goals: `:!cmd` runs cmd in the shell, like vim. The file-command allowlist stays for a colon line that does not start with `!`.
- Non-Goals: removing the `!` key. Adding mv as a file command.

## Decisions

- `colonShell` strips one leading `!` and the space around the rest. `execShell` is the same terminal handoff the `!` key uses.
- The status after the command is the shell text, without the leading bang.
- The shortcut list stays one screen. The colon row names `!cmd` instead of adding a row.

## Risks / Trade-offs

- `:!` runs whatever the operator types. The working directory is the directory on screen.
