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

### Requirement: r2d2 slack is the workspace screen
`r2d2 slack` SHALL open a screen of the workspaces stored for `c3po slack`. `r2d2 slack help` SHALL exit 0 and SHALL name `r2d2 slack`, workspace switching, write, reply, and mark read. `r2d2 slack version` SHALL print the c3po version string and SHALL exit 0. An extra argument SHALL print the screen help and SHALL exit 2. A non-terminal SHALL NOT open the screen and SHALL NOT write a session. The screen SHALL NOT print the user token.

The workspace list SHALL show each preset id and name. `j` and `k` SHALL move. `h` and `l` SHALL switch workspace on the workspace list and on the conversation list, and SHALL NOT wrap. Enter on a workspace SHALL open its conversations. Enter on a conversation SHALL open its messages, oldest first, with the cursor on the newest. Opening a conversation SHALL NOT mark it read.

`i` SHALL write a new message in the open conversation. `r` SHALL reply in the thread of the selected message. Enter on an empty draft SHALL NOT send. `m` SHALL mark the selected conversation read. `q` SHALL leave the screen. `?` SHALL toggle the shortcut list. While that list is open, `j`, enter, `i`, `r`, and `m` SHALL NOT move, open, write, reply, or mark, Esc SHALL close the list, and `q` SHALL still leave the screen. The listing SHALL NOT show shortcut text while the list is closed.

An empty account file SHALL say `no workspaces`. An empty conversation list SHALL say `no conversations`. The stored place SHALL name the screen `slack` and the open workspace, and SHALL NOT contain the user token. `--resume <id>` SHALL restore that workspace and row.

#### Scenario: Help names the screen
- **WHEN** the operator runs `r2d2 slack help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL contain `r2d2 slack`
- **AND** the text SHALL say that opening a conversation does not mark it read

#### Scenario: Enter opens messages and does not mark them read
- **WHEN** the operator opens a workspace and then a conversation
- **THEN** the screen SHALL show that conversation's messages
- **AND** it SHALL NOT mark the conversation read

#### Scenario: Write and reply send
- **WHEN** the operator presses `i`, types a message, and presses enter in an open conversation
- **THEN** the screen SHALL post that message in the conversation
- **AND** `r` SHALL post a reply on the selected message's thread

#### Scenario: Mark read is explicit
- **WHEN** the operator presses `m` on a conversation
- **THEN** the conversation SHALL be marked read
- **AND** the status SHALL say `marked`

#### Scenario: Question mark blocks the keys
- **WHEN** the shortcut list is open and the operator presses `j` and then enter
- **THEN** the cursor SHALL stay on the same workspace
- **AND** no conversation SHALL open

#### Scenario: The token stays off the screen and out of the session
- **GIVEN** a workspace whose account has a user token
- **WHEN** the workspace list is shown and the screen stores its place
- **THEN** the view SHALL NOT contain the token
- **AND** the stored place SHALL NOT contain the token

#### Scenario: An empty account file does not open Slack
- **WHEN** no workspace is configured and the operator presses enter
- **THEN** the screen SHALL say `no workspaces`
- **AND** it SHALL NOT call Slack

