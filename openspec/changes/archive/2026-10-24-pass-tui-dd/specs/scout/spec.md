## MODIFIED Requirements

### Requirement: The first argument selects an app
`c3po ast` SHALL run the ast tool with the remaining arguments. `c3po deport`, `c3po gcal`, `c3po gdrive`, `c3po gmail`, `c3po looptab`, `c3po pass`, `c3po slack`, and `c3po tunnel` SHALL run those tools with the remaining arguments. `c3po tui <screen>` SHALL be the only command that opens a terminal screen. The screens SHALL be `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, and `looptab`. `c3po tui` with no screen SHALL list those screens and SHALL exit 0. An unknown screen SHALL exit 2 and SHALL list the screens. `c3po` with no tool SHALL list `ast`, `deport`, `gcal`, `gdrive`, `gmail`, `looptab`, `pass`, `slack`, `tunnel`, and `tui`, and SHALL exit 0. An unknown command SHALL exit 2 and SHALL list the apps. The names `complexity`, `hide`, `visualize`, `vibetree`, `scout`, `cyclotree`, `l`, `o`, and `xl` SHALL be unknown commands. `c3po tunnel open` SHALL exit 2.

#### Scenario: c3po lists the apps
- **WHEN** the operator runs `c3po`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `ast`, `deport`, `gcal`, `gdrive`, `gmail`, `looptab`, `pass`, `slack`, `tunnel`, and `tui`
- **AND** the text SHALL NOT list `complexity`, `hide`, or `vibetree`
- **AND** the text SHALL NOT list `cyclotree`, `l`, `o`, or `xl` as top-level apps

#### Scenario: Ast help is reached directly
- **GIVEN** the c3po binary
- **WHEN** the operator runs `c3po ast help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name the patch command

#### Scenario: Cyclotree help is reached through tui
- **WHEN** the operator runs `c3po tui cyclotree help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL describe `j`, `k`, `h`, and `l`

#### Scenario: A direct screen name is rejected
- **WHEN** the operator runs `c3po cyclotree`
- **THEN** the command SHALL exit 2
- **AND** the same rejection SHALL apply to `l`, `o`, and `xl`
- **AND** `c3po tunnel open` SHALL exit 2

#### Scenario: tui lists the screens
- **WHEN** the operator runs `c3po tui`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, and `looptab`

#### Scenario: Drive browse starts from tui gdrive
- **WHEN** the operator runs `c3po tui gdrive help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name a preset and browse
- **AND** the text SHALL say enter opens the item in the browser
- **AND** the text SHALL say u uploads into the open folder
- **AND** the text SHALL say dd deletes the selected file
- **AND** `c3po gdrive 1 browse` SHALL exit 2

#### Scenario: Pass browse starts from tui pass
- **WHEN** the operator runs `c3po tui pass help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name a vault and an item
- **AND** the text SHALL say enter copies the selected item
- **AND** the text SHALL say the secret is not printed
- **AND** the text SHALL say dd deletes the selected item

#### Scenario: Deport lists safe external ports
- **WHEN** the operator runs `c3po tui deport help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name ports 8000 through 8999
- **AND** the text SHALL say j and k move
- **AND** the text SHALL say dd frees the selected port

#### Scenario: The spreadsheet is the xl screen
- **WHEN** the operator runs `c3po tui xl help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name an xlsx file
- **AND** `c3po xl` SHALL exit 2

#### Scenario: Looptab runs from the screen
- **WHEN** the operator runs `c3po tui looptab help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL say enter runs the selected job now

#### Scenario: Looptab help names vim
- **WHEN** the operator runs `c3po looptab help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL say in vim

#### Scenario: An unknown screen is rejected
- **WHEN** the operator runs `c3po tui nope`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL list the screens

#### Scenario: A retired name is rejected
- **WHEN** the operator runs `c3po complexity`
- **THEN** the command SHALL exit 2
- **AND** the same rejection SHALL apply to `hide`, `visualize`, `vibetree`, and `scout`

#### Scenario: An unknown command is rejected
- **WHEN** the operator runs `c3po nope`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL list the known apps, `tui` among them
