## MODIFIED Requirements

### Requirement: The first argument selects an app
`c3po ast` SHALL run the ast tool with the remaining arguments. `c3po clippy`, `c3po deport`, `c3po gates`, `c3po gcal`, `c3po gdrive`, `c3po gmail`, `c3po idealake`, `c3po looptab`, `c3po mess`, `c3po notify`, `c3po pass`, `c3po slack`, and `c3po tunnel` SHALL run those tools with the remaining arguments. There is no `c3po tui` command. `c3po tui` SHALL exit 2. `c3po` SHALL NOT open a terminal screen. Agents use the c3po commands and do not open a screen. `r2d2` is the terminal program, and running it opens the screen menu. The screens in this module SHALL be `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, and `xl`, and `r2d2 <screen>` SHALL open them, including `r2d2 <screen> --resume <id>`. `r2d2 looptab` SHALL be an r2d2 screen that calls the exported looptab functions, and the c3po screen package SHALL NOT open it. `c3po looptab` SHALL NOT open vim or any other terminal screen. `c3po` with no tool SHALL list `ast`, `clippy`, `deport`, `gates`, `gcal`, `gdrive`, `gmail`, `idealake`, `looptab`, `mess`, `notify`, `pass`, `slack`, and `tunnel`, and SHALL exit 0. It SHALL NOT list `tui`. An unknown command SHALL exit 2 and SHALL list the apps. The names `complexity`, `hide`, `visualize`, `vibetree`, `scout`, `cyclotree`, `l`, `o`, `xl`, `nlp`, and `tui` SHALL be unknown commands. `c3po tunnel open` SHALL exit 2. `c3po tui <screen>` and `c3po tui <screen> --resume <id>` SHALL exit 2.

#### Scenario: c3po lists the apps
- **WHEN** the operator runs `c3po`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `ast`, `clippy`, `deport`, `gates`, `gcal`, `gdrive`, `gmail`, `idealake`, `looptab`, `mess`, `notify`, `pass`, `slack`, and `tunnel`
- **AND** the text SHALL NOT list `tui`
- **AND** the text SHALL NOT list `complexity`, `hide`, or `vibetree`
- **AND** the text SHALL NOT list `cyclotree`, `l`, `o`, `xl`, or `nlp` as top-level apps

#### Scenario: Ast help is reached directly
- **GIVEN** the c3po binary
- **WHEN** the operator runs `c3po ast help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name the patch command

#### Scenario: c3po tui is not a command
- **WHEN** the operator runs `c3po tui cyclotree help`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL say unknown command

#### Scenario: A direct screen name is rejected
- **WHEN** the operator runs `c3po cyclotree`
- **THEN** the command SHALL exit 2
- **AND** the same rejection SHALL apply to `l`, `o`, `xl`, and `nlp`
- **AND** `c3po tunnel open` SHALL exit 2

#### Scenario: c3po tui with no screen is rejected
- **WHEN** the operator runs `c3po tui`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL say unknown command

#### Scenario: Drive browse is not a c3po command
- **WHEN** the operator runs `c3po tui gdrive help`
- **THEN** the command SHALL exit 2
- **AND** `c3po gdrive 1 browse` SHALL exit 2

#### Scenario: Pass is not a c3po screen command
- **WHEN** the operator runs `c3po tui pass help`
- **THEN** the command SHALL exit 2

#### Scenario: Deport is not a c3po screen command
- **WHEN** the operator runs `c3po tui deport help`
- **THEN** the command SHALL exit 2

#### Scenario: The spreadsheet is not a c3po command
- **WHEN** the operator runs `c3po tui xl help`
- **THEN** the command SHALL exit 2
- **AND** `c3po xl` SHALL exit 2

#### Scenario: Looptab screen is not a c3po command
- **WHEN** the operator runs `c3po tui looptab help`
- **THEN** the command SHALL exit 2

#### Scenario: NLP is not a c3po screen
- **WHEN** the operator runs `c3po tui nlp`
- **THEN** the command SHALL exit 2
- **AND** `c3po nlp` SHALL exit 2

#### Scenario: Looptab help names vim
- **WHEN** the operator runs `c3po looptab help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL say the jobs file is edited with `r2d2 looptab`
- **AND** the text SHALL NOT name vim

#### Scenario: An unknown screen name is still an unknown command
- **WHEN** the operator runs `c3po tui nope`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL say unknown command

#### Scenario: A retired name is rejected
- **WHEN** the operator runs `c3po complexity`
- **THEN** the command SHALL exit 2
- **AND** the same rejection SHALL apply to `hide`, `visualize`, `vibetree`, and `scout`

#### Scenario: An unknown command is rejected
- **WHEN** the operator runs `c3po nope`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL list the known apps
- **AND** the text SHALL NOT list `tui`

#### Scenario: Resume is not a c3po command

- **WHEN** the operator runs `c3po tui o --resume <id>`
- **THEN** the command SHALL exit 2

#### Scenario: Screen help is not a c3po command

- **WHEN** the operator runs `c3po tui o help`
- **THEN** the command SHALL exit 2
