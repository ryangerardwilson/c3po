## MODIFIED Requirements

### Requirement: Shortcuts appear only while question mark is open
An interactive screen SHALL NOT paint shortcut text on its listing. Shortcut text SHALL appear only after the operator presses `?`, and `?` SHALL toggle that list. While the list is open, a key that would move, open, copy, delete, add, or remove SHALL NOT do so, and Esc SHALL close the list. On the navigator, deport, Drive browse, the Drive preset list, pass, the tunnel route list, and the looptab job list, `q` SHALL still leave the screen. On the spreadsheet, `q` while the list is open SHALL close the list, and the sheet SHALL leave only through `:q!`. On cyclotree, `q` SHALL leave the screen. On nlp, only `?` opens or closes the shortcut list, and `q` on that list SHALL NOT leave the screen.

The navigator, deport, Drive browse, the Drive preset list, pass, the tunnel route list, the looptab job list, the spreadsheet, cyclotree, and nlp SHALL follow this. `c3po tui l` SHALL NOT paint a shortcut line. The cyclotree call chain and the focused path SHALL stay at the bottom of that screen. The spreadsheet statistics line SHALL stay. A command prompt, a filter prompt, a route name field, a route port field, a route form's save line, and a removal confirmation SHALL stay.

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

### Requirement: A screen can resume
Each screen `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, `looptab`, and `nlp` SHALL store its place in the sqlx database `c3po` when it is opened on a terminal or with `--resume <id>`. The stored place SHALL include the screen, the workspace directory, and the cursor or path the operator had open. It MUST NOT include a pass secret. `--resume <id>` SHALL restore that place. A pipe with no `--resume` id SHALL NOT write a session. Closing the screen SHALL drop the active record and SHALL keep the stored place. A missing id, a bad id, or `--resume` given twice SHALL fail without opening the screen.

#### Scenario: Resume returns to the stored place

- **GIVEN** screen `o` stored a directory and a selected path
- **WHEN** the operator runs `c3po tui o --resume <id>`
- **THEN** the navigator SHALL open in that directory at that path

#### Scenario: A pipe does not write a session

- **GIVEN** stdout is not a terminal
- **WHEN** `c3po tui l` runs with no `--resume` id
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
