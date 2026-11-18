## ADDED Requirements

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
