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

### Requirement: The slack screen snapshots its place
While `r2d2 slack` is open it SHALL publish the c3po database about every 10 minutes, and it SHALL publish once when the screen leaves. That publish SHALL wait until the upload finishes or 60 seconds pass. A key that stores the place SHALL NOT publish. The stored place SHALL name the screen `slack` and SHALL NOT contain the user token.

When the runtime database is missing, the screen SHALL load the last published backup before it reads the place. When that load fails for a reason other than the object being absent, the screen SHALL NOT open and SHALL NOT write a session. `--resume <id>` SHALL restore the workspace and row from the loaded database.

#### Scenario: Moving does not publish
- **WHEN** the operator moves the cursor on the slack screen
- **THEN** the place SHALL be stored in the runtime database
- **AND** that key SHALL NOT publish the backup

#### Scenario: Leaving publishes once
- **WHEN** the operator leaves the slack screen
- **THEN** the screen SHALL publish the c3po database once
- **AND** it SHALL wait for that publish

#### Scenario: Resume uses the loaded backup
- **GIVEN** the runtime file was missing and the backup holds a slack place
- **WHEN** the operator runs `r2d2 slack --resume <id>`
- **THEN** the screen SHALL open that workspace and row
- **AND** the place SHALL NOT contain the user token

### Requirement: r2d2 slack is the desk
`r2d2 slack` SHALL open the Slack desk for the workspaces on that desk. `r2d2 slack help` SHALL exit 0 and SHALL name `r2d2 slack`, workspace switching, and mark read. The help SHALL say that opening an item does not mark it read. `r2d2 slack version` SHALL print the c3po version string and SHALL exit 0. An extra argument SHALL print the screen help and SHALL exit 2. A non-terminal SHALL NOT open the screen and SHALL NOT write a session. The screen SHALL NOT print the user token. The screen SHALL NOT post a message. `s` SHALL ask the desk to refresh from Slack and SHALL then reload the table. While that refresh runs the status SHALL say `syncing`. A failed refresh SHALL leave the rows in place and SHALL show the failure.

The heading SHALL name each desk workspace, and the current workspace SHALL be distinguished. `h` and `l` SHALL switch workspace and SHALL wrap. The table SHALL show the sections `DMS`, `TAGGED`, and `SHOULD KNOW`, each with its count. A row SHALL show the item's time, then the sender or else the conversation, then the desk's reason when that reason is set and otherwise the message text. When the desk has a refresh time, the screen SHALL say `last synced at` and SHALL show that time. `j` and `k` SHALL move among those rows and SHALL wrap. On the last row, `j` SHALL move to the first row. On the first row, `k` SHALL move to the last row. An empty table SHALL leave the cursor in place. Enter SHALL open the selected row through the desk and SHALL NOT mark it read. A successful open SHALL leave the screen. `r` SHALL mark the selected row read on the desk, SHALL reload the desk, and SHALL drop that row. The status SHALL say `marked`. A failed open or a failed mark SHALL leave the row in place and SHALL show the failure. `q` SHALL leave the screen. `?` SHALL toggle the shortcut list. While that list is open, `j`, enter, `r`, and `s` SHALL NOT move, open, mark, or sync, Esc SHALL close the list, and `q` SHALL still leave the screen. The table SHALL NOT show shortcut text while the list is closed.

A desk with no workspace and no refresh time SHALL say `no brief yet`. A workspace with no rows SHALL list each section with a count of 0 and SHALL say `no items`. The screen SHALL NOT say `watching after`. A failed desk load SHALL show that failure and SHALL NOT list rows from a Slack search. The stored place SHALL name the screen `slack` and the open workspace, and SHALL NOT contain the user token. `--resume <id>` SHALL restore that workspace and row.

#### Scenario: Help names the desk
- **WHEN** the operator runs `r2d2 slack help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL contain `r2d2 slack`
- **AND** the text SHALL say that opening an item does not mark it read

#### Scenario: The table shows the three desk sections
- **WHEN** the desk has a direct message, a tagged message, and a should-know message with a reason
- **THEN** the screen SHALL show `DMS`, `TAGGED`, and `SHOULD KNOW` with those counts
- **AND** the should-know row SHALL show the reason
- **AND** the row SHALL show the time and the sender

#### Scenario: Enter opens the row and does not mark it read
- **WHEN** the operator presses enter on a row
- **THEN** the desk SHALL open that row
- **AND** the row SHALL NOT be marked read
- **AND** the screen SHALL leave

#### Scenario: Mark read drops the row
- **WHEN** the operator presses `r` on a row
- **THEN** the desk SHALL mark that row read
- **AND** the row SHALL leave the table
- **AND** the status SHALL say `marked`

#### Scenario: Workspace switching wraps
- **WHEN** the operator presses `l` on the last workspace
- **THEN** the screen SHALL show the first workspace's rows

#### Scenario: j and k wrap the rows
- **WHEN** the desk table has three rows and the cursor is on the last row
- **AND** the operator presses `j`
- **THEN** the cursor SHALL be on the first row
- **AND** `k` SHALL move it back to the last row

#### Scenario: Question mark blocks the keys
- **WHEN** the shortcut list is open and the operator presses `j`, then enter, then `r`, then `s`
- **THEN** the cursor SHALL stay on the same row
- **AND** the row SHALL NOT open
- **AND** the row SHALL NOT be marked read
- **AND** the desk SHALL NOT refresh

#### Scenario: The token stays off the screen and out of the session
- **GIVEN** a desk workspace
- **WHEN** the table is shown and the screen stores its place
- **THEN** the view SHALL NOT contain the user token
- **AND** the stored place SHALL NOT contain the user token

#### Scenario: An empty workspace says no items
- **GIVEN** a workspace with no rows and an intake time
- **WHEN** the table is shown
- **THEN** the screen SHALL say `no items`
- **AND** the screen SHALL NOT say `watching after`
- **AND** each section count SHALL be 0

#### Scenario: The screen shows when the desk last synced
- **GIVEN** the desk has a refresh time
- **WHEN** the table is shown
- **THEN** the screen SHALL say `last synced at`
- **AND** the screen SHALL show that refresh time

#### Scenario: Sync reloads the desk
- **WHEN** the operator presses `s`
- **THEN** the desk SHALL refresh from Slack
- **AND** the table SHALL show the refreshed rows
- **AND** the screen SHALL say `last synced at`

#### Scenario: A failed sync leaves the table
- **WHEN** the desk refresh fails
- **THEN** the rows SHALL stay
- **AND** the screen SHALL show the failure

#### Scenario: A missing desk does not search Slack
- **WHEN** the desk cannot be read
- **THEN** the screen SHALL show that failure
- **AND** it SHALL NOT search Slack

