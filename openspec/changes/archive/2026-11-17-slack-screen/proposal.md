# Slack screen

## Why

Slack on this machine is an agent CLI, `c3po slack`, and a Quickshell desk. The terminal program should open Slack as a screen, the same way it opens Drive or pass.

## What Changes

- `r2d2 slack` lists the workspaces in the c3po slack account file.
- `h` and `l` switch workspace. Enter opens a conversation, then its messages.
- `i` writes in the open conversation. `r` replies in that thread. `m` marks the conversation read.
- Opening a conversation does not mark it read.
- Shortcut text stays behind `?`.
- The screen stores its place and does not store the user token.
- `c3po slack` stays the agent CLI. The Quickshell desk is unchanged.

## Impact

- Specs: `openspec/specs/slack/spec.md`, `openspec/specs/tui/spec.md`
- Code: `slack/`, `screen/`
- r2d2 gains the screen name `slack` and calls this package.
