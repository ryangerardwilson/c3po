# tui Specification

## Purpose
Keeps shortcut text off every interactive c3po screen until the operator presses question mark.
## Requirements
### Requirement: Shortcuts appear only while question mark is open
An interactive screen SHALL NOT paint shortcut text on its listing. Shortcut text SHALL appear only after the operator presses `?`, and `?` SHALL toggle that list. While the list is open, a key that would move, open, copy, delete, add, or remove SHALL NOT do so, and Esc SHALL close the list. On the navigator, deport, Drive browse, the Drive preset list, pass, the tunnel route list, the looptab job list, and the slack desk, `q` SHALL still leave the screen. On the spreadsheet, `q` while the list is open SHALL close the list, and the sheet SHALL leave only through `:q!`. On cyclotree, `q` SHALL leave the screen. On nlp, only `?` opens or closes the shortcut list, and `q` on that list SHALL NOT leave the screen.

The navigator, deport, Drive browse, the Drive preset list, pass, the tunnel route list, the looptab job list, the spreadsheet, cyclotree, nlp, and the slack desk SHALL follow this. `r2d2 l` SHALL NOT paint a shortcut line. The cyclotree call chain and the focused path SHALL stay at the bottom of that screen. The spreadsheet statistics line SHALL stay. A command prompt, a filter prompt, a route name field, a route port field, a route form's save line, and a removal confirmation SHALL stay.

#### Scenario: The navigator hides the legend
- **WHEN** the navigator is listing a directory and the operator has not pressed `?`
- **THEN** the view SHALL NOT contain `j/k move`

#### Scenario: Question mark lists the navigator shortcuts
- **WHEN** the operator presses `?` on the navigator and then presses `?` again
- **THEN** the first view SHALL name cut, copy, paste, and quit
- **AND** the second view SHALL show the directory listing

#### Scenario: Deport hides the legend until question mark
- **WHEN** the operator presses `?` on a deport list, then `j`, then Esc
- **THEN** the view before `?` SHALL NOT contain `j/k move`
- **AND** the open list SHALL name `dd` and quit
- **AND** `j` SHALL leave the cursor on the same port
- **AND** Esc SHALL show the port list again

#### Scenario: Drive browse hides the legend until question mark
- **WHEN** the operator presses `?` on a Drive folder, then `j`, then Esc
- **THEN** the view before `?` SHALL NOT contain `j/k move`
- **AND** the open list SHALL name upload and delete
- **AND** `j` SHALL leave the cursor on the same row
- **AND** Esc SHALL show the folder again

#### Scenario: The preset list hides the legend until question mark
- **WHEN** the operator presses `?` on the Drive preset list, then `j`, then Esc
- **THEN** the view before `?` SHALL NOT contain `j/k move`
- **AND** the open list SHALL name select and quit
- **AND** `j` SHALL leave the cursor on the same preset
- **AND** Esc SHALL show the preset list again

#### Scenario: Pass hides the legend until question mark
- **WHEN** the operator presses `?` on the pass vault list, then `j`, then Esc
- **THEN** the view before `?` SHALL NOT contain `j/k move`
- **AND** the open list SHALL name open and copy
- **AND** the open list SHALL name `dd`
- **AND** `j` SHALL leave the cursor on the same row
- **AND** Esc SHALL show the vault list again

#### Scenario: The tunnel route list hides the legend until question mark
- **WHEN** the operator presses `?` on the tunnel route list, then `j`, then Esc
- **THEN** the view before `?` SHALL NOT contain `a add`
- **AND** the open list SHALL name add, remove, and quit
- **AND** `j` SHALL leave the cursor on the same route
- **AND** Esc SHALL show the route list again
- **AND** the add form SHALL still say `enter save` and `esc cancel`
- **AND** a removal SHALL still ask `y or n`

#### Scenario: The looptab list hides the legend until question mark
- **WHEN** the operator presses `?` on the looptab job list, then `j`, then Esc
- **THEN** the view before `?` SHALL NOT contain `j/k move`
- **AND** the open list SHALL name enter and quit
- **AND** `j` SHALL leave the cursor on the same job
- **AND** Esc SHALL show the job list again

#### Scenario: The spreadsheet keeps shortcuts behind question mark
- **WHEN** the spreadsheet is showing the grid and the operator presses `?`
- **THEN** the grid SHALL NOT show the shortcut list
- **AND** the shortcut list SHALL be shown after `?`

#### Scenario: Cyclotree keeps shortcuts behind question mark
- **WHEN** cyclotree is showing the tree and the operator has not pressed `?`
- **THEN** the tree SHALL NOT show the shortcut list
- **AND** the bottom SHALL show the call chain and the focused path

#### Scenario: NLP keeps shortcuts behind question mark
- **WHEN** the nlp course list is showing and the operator has not pressed `?`
- **THEN** the list SHALL NOT show a shortcut string
- **AND** `?` SHALL open the shortcut list
- **AND** Esc SHALL leave that list open

#### Scenario: Slack hides the legend until question mark
- **WHEN** the operator presses `?` on the slack desk, then `j`, then Esc
- **THEN** the view before `?` SHALL NOT contain `j/k move`
- **AND** the open list SHALL name workspace, open, read, and sync
- **AND** `j` SHALL leave the cursor on the same row
- **AND** Esc SHALL show the desk again

### Requirement: dd deletes the selected pass item
On `r2d2 pass`, the first `d` SHALL arm deletion and SHALL show `d`. A second `d` SHALL delete the selected item from its open vault and SHALL reload that vault. The cursor SHALL stay at the same index, clamped to the remaining rows. The status SHALL be `deleted` followed by the item title. The secret SHALL stay unread and unprinted. Esc SHALL cancel the arm and SHALL leave the screen open. Any other key SHALL clear the arm and SHALL run as that key. `?` SHALL clear the arm and SHALL open the shortcut list. While the shortcut list is open, `d` SHALL leave the item in place. A vault row SHALL stay, and the status SHALL be `dd deletes an item`. An empty item list SHALL report `no items`. An empty vault list SHALL report `no vaults`. A failed delete SHALL leave the item in place and SHALL show the failure on the status line.

#### Scenario: dd deletes the selected item
- **WHEN** the operator presses `dd` on an item in an open vault
- **THEN** that item SHALL be removed from the vault
- **AND** the listing SHALL show the remaining items
- **AND** the cursor SHALL stay at the same index
- **AND** the status SHALL name the deleted title
- **AND** the view SHALL NOT contain the secret

#### Scenario: Another key cancels the arm
- **WHEN** the operator presses `d` and then `j` on an item list
- **THEN** the item SHALL remain
- **AND** the cursor SHALL move

#### Scenario: Esc cancels the arm
- **WHEN** the operator presses `d` and then Esc on an item
- **THEN** the item SHALL remain
- **AND** the screen SHALL stay open

#### Scenario: dd leaves a vault
- **WHEN** the operator presses `dd` on a vault row
- **THEN** the vault SHALL remain
- **AND** the status SHALL be `dd deletes an item`

#### Scenario: An empty item list reports no items
- **WHEN** the operator presses `dd` in a vault that has no items
- **THEN** the status SHALL be `no items`

#### Scenario: An empty vault list reports no vaults
- **WHEN** the operator presses `dd` and no vaults are listed
- **THEN** the status SHALL be `no vaults`

#### Scenario: A failed delete leaves the item
- **WHEN** deleting the selected item fails
- **THEN** the item SHALL remain
- **AND** the status SHALL report the failure
- **AND** the view SHALL NOT contain the secret

#### Scenario: Help ignores d
- **WHEN** the shortcut list is open and the operator presses `d`
- **THEN** the item SHALL remain
- **AND** the shortcut list SHALL stay open

### Requirement: A screen can resume
Each screen `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, `looptab`, `nlp`, and `slack` SHALL store its place in the sqlx database `c3po` when it is opened on a terminal or with `--resume <id>`. The stored place SHALL include the screen, the workspace directory, and the cursor or path the operator had open. It MUST NOT include a pass secret. `--resume <id>` SHALL restore that place. A pipe with no `--resume` id SHALL NOT write a session. Closing the screen SHALL drop the active record and SHALL keep the stored place. A missing id, a bad id, or `--resume` given twice SHALL fail without opening the screen.

#### Scenario: Resume returns to the stored place

- **GIVEN** screen `o` stored a directory and a selected path
- **WHEN** the operator runs `r2d2 o --resume <id>`
- **THEN** the navigator SHALL open in that directory at that path

#### Scenario: A pipe does not write a session

- **GIVEN** stdout is not a terminal
- **WHEN** `r2d2 l` runs with no `--resume` id
- **THEN** it MUST NOT write a TUI session

#### Scenario: NLP stores the course place

- **GIVEN** the nlp screen has a course selected
- **WHEN** it stores its place
- **THEN** the record MUST name the screen `nlp`
- **AND** the record MUST identify the course
- **AND** the record MUST NOT contain the course markdown

#### Scenario: Pass keeps the secret out of the session

- **GIVEN** the pass screen is showing an item
- **WHEN** it stores its place
- **THEN** the record MUST identify the vault and the item
- **AND** it MUST NOT contain the item secret

#### Scenario: Slack keeps the token out of the session

- **GIVEN** the slack screen has a workspace open
- **WHEN** it stores its place
- **THEN** the record MUST name the screen `slack`
- **AND** the record MUST identify the workspace
- **AND** it MUST NOT contain the user token

### Requirement: Screen paint is black, white, or gray
A screen SHALL paint text in black, white, or gray only. Red, green, and blue channels of one painted color SHALL be equal. A magnitude, such as a text file's line count or a binary's size, SHALL use a darker gray for a smaller value and a lighter gray for a larger value. Bold, dim, and reverse MAY mark a row or a state. A screen SHALL NOT paint cyan, yellow, red, green, or any other hue. `NO_COLOR` and a color mode of never SHALL leave that text unpainted.

#### Scenario: The directory tree scales in gray
- **WHEN** `r2d2 l` paints a text file and a binary with color on
- **THEN** each label SHALL use equal red, green, and blue channels
- **AND** a directory SHALL be white
- **AND** the tree glyphs SHALL be gray

#### Scenario: The spreadsheet cursor is gray
- **WHEN** the spreadsheet paints the focused cell with color on
- **THEN** the cell shade SHALL be black on light gray
- **AND** the cursor SHALL be white on a darker gray
- **AND** the frame SHALL NOT use an unequal red, green, and blue channel

### Requirement: A screen backup survives power loss
A screen SHALL write its place into the runtime c3po sqlite file when the place changes. That write SHALL NOT publish `s3://backups/c3po.sqlite`. A snapshot SHALL publish that file about every 10 minutes, SHALL upload a consistent copy, and SHALL wait until the upload finishes or 60 seconds pass. A database whose `record` table has no rows SHALL NOT be published. `SQLX_ROOT` SHALL NOT publish and SHALL NOT restore.

When the runtime file is missing, opening a session SHALL load `s3://backups/c3po.sqlite` and SHALL wait up to 60 seconds. When that load fails for a reason other than the object being absent, the open SHALL fail, SHALL NOT create a database, and SHALL NOT publish. When the object is absent, an empty database MAY be created and SHALL NOT be published until it has a record.

The installer SHALL enable a user timer `c3po-snapshot.timer` with `OnActiveSec=10min`, `OnUnitActiveSec=10min`, and `AccuracySec=5s`. The timer SHALL run `c3po __snapshot`. That command SHALL publish the runtime database when it has a record, and SHALL NOT create a database when the runtime file is missing. The installer SHALL NOT run a snapshot while it installs.

#### Scenario: A key does not publish
- **WHEN** a screen stores a new place
- **THEN** the runtime database SHALL contain that place
- **AND** the backup SHALL NOT be published by that write

#### Scenario: An empty database is not published
- **GIVEN** the runtime database has a `record` table with no rows
- **WHEN** a snapshot runs
- **THEN** `s3://backups/c3po.sqlite` SHALL NOT be replaced

#### Scenario: A failed restore does not create a database
- **GIVEN** the runtime file is missing and the backup load fails
- **WHEN** a session open runs
- **THEN** the open SHALL fail
- **AND** the runtime file SHALL stay missing

#### Scenario: A missing runtime file loads the backup
- **GIVEN** the runtime file is missing and the backup object exists
- **WHEN** a session opens
- **THEN** the runtime file SHALL be loaded from that object before the place is read

#### Scenario: The timer publishes every 10 minutes
- **WHEN** c3po is installed
- **THEN** `c3po-snapshot.timer` SHALL set `OnActiveSec=10min` and `OnUnitActiveSec=10min`
- **AND** its service SHALL run `c3po __snapshot`

