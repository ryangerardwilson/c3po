## Why

The Slack user CLI was removed with the old `sluser` name. The accounts and the message commands are still needed, as a c3po app named slack.

## What Changes

- `c3po slack` lists, sends, replies, edits, and reads Slack messages for a numbered preset.
- `c3po sluser` is an unknown command.
- Help and usage say `c3po slack`. The word sluser is not a command.
- Account config moves from `~/.config/sluser` to `~/.config/c3po/slack`. Event-cache state moves from `~/.local/state/sluser` to `~/.local/state/c3po/slack`.
- `~/.config/slack` and `~/.local/state/slack` stay where they are.
- `c3po slack version` prints the c3po version.

## Capabilities

### New Capabilities

- `slack`: The Slack user CLI, invoked as `c3po slack`.

### Modified Capabilities

- `scout`: The app list includes slack.

## Impact

`c3po slack` joins the app map. The event cache uses sqlite. The home Slack desk reads `~/.config/c3po/slack/config.json` when that file is present.
