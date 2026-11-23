## REMOVED Requirements

### Requirement: The slack screen snapshots its place
**Reason**: Leaving the screen waited about 6 s on a whole-database upload, and a timer published every 10 minutes from inside the screen. nebula keep now backs the c3po database up after each commit, so the screen never publishes.
**Migration**: The place is still stored as the screen moves, through the session writer; see `The slack screen stores its place and never waits on a backup`.

## ADDED Requirements

### Requirement: The slack screen stores its place and never waits on a backup
The screen SHALL store its place in the runtime c3po database when the place changes, from a background writer that keeps only the latest place, and SHALL store the last place before it leaves. Neither a key nor leaving the screen SHALL publish or wait on a backup; the backup of the c3po database follows each write through nebula keep. The stored place SHALL name the screen `slack` and SHALL NOT contain the user token.

When the runtime database is missing, the screen SHALL load it through `nebula keep restore` before it reads the place. When that load fails, the screen SHALL NOT open and SHALL NOT write a session. `--resume <id>` SHALL restore the workspace and row from the loaded database.

#### Scenario: Moving does not wait on a backup
- **WHEN** the operator moves the cursor on the slack screen
- **THEN** the place SHALL be stored in the runtime database
- **AND** that key SHALL NOT wait on a publish

#### Scenario: Leaving does not wait on a backup
- **GIVEN** no mark is pending
- **WHEN** the operator leaves the slack screen
- **THEN** the screen SHALL leave without publishing the c3po database

#### Scenario: Resume uses the loaded backup
- **GIVEN** the runtime file was missing and the backup holds a slack place
- **WHEN** the operator runs `r2d2 slack --resume <id>`
- **THEN** the screen SHALL open that workspace and row
- **AND** the place SHALL NOT contain the user token

## MODIFIED Requirements

### Requirement: r2d2 slack is the desk
`r2d2 slack` SHALL open the Slack desk for the workspaces on that desk. `r2d2 slack help` SHALL exit 0 and SHALL name `r2d2 slack`, workspace switching, and mark read. The help SHALL say that opening an item does not mark it read. The help SHALL say that the screen syncs about every 10 minutes. `r2d2 slack version` SHALL print the c3po version string and SHALL exit 0. An extra argument SHALL print the screen help and SHALL exit 2. A non-terminal SHALL NOT open the screen and SHALL NOT write a session. The screen SHALL NOT print the user token. The screen SHALL NOT post a message. `s` SHALL ask the desk to refresh from Slack off the key loop and SHALL then reload the table. While that refresh runs the status SHALL say `syncing`, and `j`, `k`, `h`, `l`, `r`, and `q` SHALL keep working. A failed refresh SHALL leave the rows in place and SHALL show the failure. While the screen stays open it SHALL ask the desk to refresh about every 10 minutes on that same path. The screen SHALL NOT refresh when it opens. While the shortcut list is open the screen SHALL NOT refresh on that interval. After the list closes, a refresh that came due during the list SHALL run. Pressing `s` SHALL start the 10 minute wait again. A failed automatic refresh SHALL wait about 10 minutes before the next one.

The heading SHALL name each desk workspace, and the current workspace SHALL be distinguished. `h` and `l` SHALL switch workspace and SHALL wrap. The table SHALL show the sections `DMS`, `TAGGED`, and `SHOULD KNOW`, each with its count. A row SHALL show the item's time, then the sender or else the conversation, then the desk's reason when that reason is set and otherwise the message text. When the desk has a refresh time, the screen SHALL say `last synced at` and SHALL show that time. `j` and `k` SHALL move among those rows and SHALL wrap. On the last row, `j` SHALL move to the first row. On the first row, `k` SHALL move to the last row. An empty table SHALL leave the cursor in place. Enter SHALL open the selected row through the desk and SHALL NOT mark it read. A successful open SHALL leave the screen. `r` SHALL drop the selected row from the table at once with the status `marking`, SHALL mark it read on the desk off the key loop, and SHALL then show the desk's answer with the status `marked`. A failed mark SHALL bring the row back and SHALL show the failure. A failed open SHALL leave the row in place and SHALL show the failure. While an open, a mark, or a refresh runs, the other keys SHALL keep working. `q` SHALL leave the screen after the marks it already sent reach the desk, waiting at most 10 seconds and saying `saving` with their count meanwhile. `?` SHALL toggle the shortcut list. While that list is open, `j`, enter, `r`, and `s` SHALL NOT move, open, mark, or sync, Esc SHALL close the list, and `q` SHALL still leave the screen. The table SHALL NOT show shortcut text while the list is closed.

A desk with no workspace and no refresh time SHALL say `no brief yet`. A workspace with no rows SHALL list each section with a count of 0 and SHALL say `no items`. The screen SHALL NOT say `watching after`. A failed desk load SHALL show that failure and SHALL NOT list rows from a Slack search. The stored place SHALL name the screen `slack` and the open workspace, and SHALL NOT contain the user token. `--resume <id>` SHALL restore that workspace and row.

#### Scenario: Help names the desk
- **WHEN** the operator runs `r2d2 slack help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL contain `r2d2 slack`
- **AND** the text SHALL say that opening an item does not mark it read
- **AND** the text SHALL say that the screen syncs about every 10 minutes

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
- **THEN** the row SHALL leave the table before the desk answers
- **AND** the desk SHALL mark that row read
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

#### Scenario: The open screen syncs about every 10 minutes
- **WHEN** the screen stays open for about 10 minutes
- **THEN** the desk SHALL refresh from Slack
- **AND** the status SHALL say `syncing` before the refresh returns
- **AND** the table SHALL show the refreshed rows
- **AND** the screen SHALL say `last synced at`

#### Scenario: Opening the screen does not sync
- **WHEN** the operator opens the screen and quits before 10 minutes
- **THEN** the desk SHALL NOT refresh

#### Scenario: The shortcut list holds the refresh
- **WHEN** the shortcut list stays open past the 10 minute interval
- **THEN** the desk SHALL NOT refresh
- **AND** after the list closes the desk SHALL refresh

#### Scenario: A failed automatic sync leaves the table
- **WHEN** the automatic refresh fails
- **THEN** the rows SHALL stay
- **AND** the screen SHALL show the failure
- **AND** the screen SHALL wait about 10 minutes before refreshing again

#### Scenario: s starts the 10 minute wait again
- **WHEN** the operator presses `s` before the interval elapses
- **THEN** the next automatic refresh SHALL wait about 10 minutes from that sync

#### Scenario: A missing desk does not search Slack
- **WHEN** the desk cannot be read
- **THEN** the screen SHALL show that failure
- **AND** it SHALL NOT search Slack
