# slack Specification

## Purpose
Slack user commands for numbered presets. The command is c3po slack, and it keeps user tokens and message history on this machine.

## Requirements
### Requirement: The command is c3po slack
The Slack user CLI SHALL be `c3po slack`. `c3po slack` with no arguments and `c3po slack help` SHALL print the same help and SHALL exit 0. That help SHALL contain `c3po slack` and SHALL NOT contain `sluser`. `c3po slack version` SHALL print the c3po version string and SHALL exit 0. Extra arguments after `help` or `version` SHALL exit 2. `c3po sluser` SHALL exit 2. An unknown slack command SHALL exit 2. A successful command SHALL exit 0. A runtime failure SHALL exit 1.

#### Scenario: Help names slack
- **WHEN** the operator runs `c3po slack`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL contain `c3po slack`
- **AND** the text SHALL NOT contain `sluser`
- **AND** `c3po slack help` SHALL print the same text

#### Scenario: Version is the c3po version
- **WHEN** the operator runs `c3po slack version`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL be the c3po version string

#### Scenario: The retired name is rejected
- **WHEN** the operator runs `c3po sluser`
- **THEN** the command SHALL exit 2

### Requirement: Accounts use user tokens under the c3po config
The account file SHALL be `$XDG_CONFIG_HOME/c3po/slack/config.json` when `XDG_CONFIG_HOME` is set, and `~/.config/c3po/slack/config.json` otherwise. Event-cache state SHALL be `$XDG_STATE_HOME/c3po/slack` when `XDG_STATE_HOME` is set, and `~/.local/state/c3po/slack` otherwise. When the c3po slack config directory is absent and `~/.config/sluser` is present, the first run SHALL rename that directory to `~/.config/c3po/slack`. When the c3po slack state directory is absent and `~/.local/state/sluser` is present, the first run SHALL rename that directory to `~/.local/state/c3po/slack`. `~/.config/slack` and `~/.local/state/slack` SHALL be left in place. A preset SHALL store a user token at `token.user`. A token that starts with `xoxb-` SHALL be rejected. `c3po slack auth <preset> import` SHALL read `~/.openclaw/credentials/slack-user-token`. Help SHALL NOT print an absolute config path. The account file SHALL be written with mode 0600.

#### Scenario: Auth writes the user token under c3po
- **GIVEN** `XDG_CONFIG_HOME` points at an empty config directory
- **WHEN** the operator runs `c3po slack auth 2 user xoxp-user name work`
- **THEN** the command SHALL exit 0
- **AND** `c3po/slack/config.json` under that directory SHALL contain preset `2` with `token.user`
- **AND** the file mode SHALL be `0600`

#### Scenario: A bot token is rejected
- **WHEN** the operator runs `c3po slack auth 1 user xoxb-bot`
- **THEN** the command SHALL exit 2
- **AND** the error SHALL say the command does not accept bot tokens

#### Scenario: Official slack directories are not adopted
- **GIVEN** config exists at `~/.config/slack` and the c3po slack config is absent
- **WHEN** c3po runs
- **THEN** `~/.config/slack` SHALL remain in place

### Requirement: Messages keep the slack grammar
`c3po slack <preset> list` SHALL list messages for that preset. `c3po slack <preset> unread count` SHALL report the pending notification total. `c3po slack <preset> send to <target> body <message>` SHALL be the send form. `c3po slack <preset> reply to <message_id> body <message>` SHALL be the reply form. `c3po slack config` SHALL print a redacted summary that does not include the token. `c3po slack accounts list` SHALL list presets. A command that names an account SHALL require a preset when any preset exists.

#### Scenario: Reply names the message
- **WHEN** help is printed
- **THEN** the text SHALL contain `c3po slack <preset> reply to <message_id>`
- **AND** the text SHALL contain `c3po slack <preset> unread count`
- **AND** the text SHALL contain `c3po slack config`

