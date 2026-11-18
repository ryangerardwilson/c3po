## MODIFIED Requirements

### Requirement: r2d2 slack is the desk
`r2d2 slack` SHALL open the Slack desk for the workspaces on that desk. `r2d2 slack help` SHALL exit 0 and SHALL name `r2d2 slack`, workspace switching, and mark read. The help SHALL say that opening an item does not mark it read. `r2d2 slack version` SHALL print the c3po version string and SHALL exit 0. An extra argument SHALL print the screen help and SHALL exit 2. A non-terminal SHALL NOT open the screen and SHALL NOT write a session. The screen SHALL NOT print the user token. The screen SHALL NOT post a message. `s` SHALL ask the desk to refresh from Slack and SHALL then reload the table. While that refresh runs the status SHALL say `syncing`. A failed refresh SHALL leave the rows in place and SHALL show the failure.

The heading SHALL name each desk workspace, and the current workspace SHALL be distinguished. `h` and `l` SHALL switch workspace and SHALL wrap. The table SHALL show the sections `DMS`, `TAGGED`, and `SHOULD KNOW`, each with its count. A row SHALL show the item's time, then the sender or else the conversation, then the desk's reason when that reason is set and otherwise the message text. When the desk has a refresh time, the screen SHALL say `last synced at` and SHALL show that time. `j` and `k` SHALL move among those rows and SHALL NOT wrap. Enter SHALL open the selected row through the desk and SHALL NOT mark it read. A successful open SHALL leave the screen. `r` SHALL mark the selected row read on the desk, SHALL reload the desk, and SHALL drop that row. The status SHALL say `marked`. A failed open or a failed mark SHALL leave the row in place and SHALL show the failure. `q` SHALL leave the screen. `?` SHALL toggle the shortcut list. While that list is open, `j`, enter, `r`, and `s` SHALL NOT move, open, mark, or sync, Esc SHALL close the list, and `q` SHALL still leave the screen. The table SHALL NOT show shortcut text while the list is closed.

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
