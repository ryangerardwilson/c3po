## MODIFIED Requirements

### Requirement: The first argument selects an app
`c3po ast` SHALL run the ast tool with the remaining arguments. `c3po clippy`, `c3po deport`, `c3po dirt`, `c3po gates`, `c3po gcal`, `c3po gdrive`, `c3po gmail`, `c3po idealake`, `c3po looptab`, `c3po mess`, `c3po notify`, `c3po pass`, `c3po slack`, and `c3po tunnel` SHALL run those tools with the remaining arguments. There is no `c3po tui` command. `c3po tui` SHALL exit 2. `c3po` SHALL NOT open a terminal screen. Agents use the c3po commands and do not open a screen. `r2d2` is the terminal program, and running it opens the screen menu. The screens in this module SHALL be `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, and `xl`, and `r2d2 <screen>` SHALL open them, including `r2d2 <screen> --resume <id>`. `r2d2 looptab` SHALL be an r2d2 screen that calls the exported looptab functions, and the c3po screen package SHALL NOT open it. `c3po looptab` SHALL NOT open vim or any other terminal screen. `c3po` with no tool SHALL list `ast`, `clippy`, `deport`, `dirt`, `gates`, `gcal`, `gdrive`, `gmail`, `idealake`, `looptab`, `mess`, `notify`, `pass`, `slack`, and `tunnel`, and SHALL exit 0. It SHALL NOT list `tui`. An unknown command SHALL exit 2 and SHALL list the apps. The names `complexity`, `hide`, `visualize`, `vibetree`, `scout`, `cyclotree`, `l`, `o`, `xl`, `nlp`, and `tui` SHALL be unknown commands. `c3po tunnel open` SHALL exit 2. `c3po tui <screen>` and `c3po tui <screen> --resume <id>` SHALL exit 2.

#### Scenario: c3po lists the apps
- **WHEN** the operator runs `c3po`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `ast`, `clippy`, `deport`, `dirt`, `gates`, `gcal`, `gdrive`, `gmail`, `idealake`, `looptab`, `mess`, `notify`, `pass`, `slack`, and `tunnel`
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

#### Scenario: Looptab help names the screen
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

### Requirement: One help command
`c3po help` SHALL exit 0, SHALL show `c3po help -f`, and SHALL list `ast`, `clippy`, `deport`, `dirt`, `gates`, `gcal`, `gdrive`, `gmail`, `idealake`, `looptab`, `mess`, `notify`, `pass`, `slack`, and `tunnel`. It SHALL NOT list `tui`. `c3po help -f <category>` SHALL explain that category. `c3po help -f ast <feature>` SHALL explain that ast command. `c3po help -f <app>` SHALL print the same text as `c3po <app> help`. An unknown category SHALL exit 2. An unknown ast feature SHALL exit 2. A feature after an app other than ast SHALL exit 2. `c3po help -f tui`, `c3po help -f tui cyclotree`, `c3po help -f vibetree`, `c3po help -f cyclotree`, `c3po help -f l`, `c3po help -f o`, and `c3po help -f nlp` SHALL exit 2.

#### Scenario: Help shows the single form
- **WHEN** the operator runs `c3po help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL show `c3po help -f`
- **AND** the text SHALL list `ast`, `clippy`, `deport`, `dirt`, `gates`, `gcal`, `gdrive`, `gmail`, `idealake`, `looptab`, `mess`, `notify`, `pass`, `slack`, and `tunnel`
- **AND** the text SHALL NOT list `tui`
- **AND** the text SHALL NOT list `vibetree`

#### Scenario: tui is not a help category
- **WHEN** the operator runs `c3po help -f tui cyclotree`
- **THEN** the command SHALL exit 2

#### Scenario: Spreadsheet help is not a c3po category
- **WHEN** the operator runs `c3po help -f tui xl`
- **THEN** the command SHALL exit 2

#### Scenario: Looptab screen help is not a c3po category
- **WHEN** the operator runs `c3po help -f tui looptab`
- **THEN** the command SHALL exit 2

#### Scenario: NLP help is not a c3po screen
- **WHEN** the operator runs `c3po help -f tui nlp`
- **THEN** the command SHALL exit 2

#### Scenario: A feature explains one ast command
- **WHEN** the operator runs `c3po help -f ast patch`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL describe patch

#### Scenario: An unknown category is rejected
- **WHEN** the operator runs `c3po help -f nope`
- **THEN** the command SHALL exit 2
- **AND** `c3po help -f vibetree` SHALL exit 2
- **AND** `c3po help -f cyclotree`, `c3po help -f l`, `c3po help -f o`, and `c3po help -f nlp` SHALL exit 2

#### Scenario: An unknown feature is rejected
- **WHEN** the operator runs `c3po help -f ast nope`
- **THEN** the command SHALL exit 2
- **AND** `c3po help -f tui nope` SHALL exit 2
- **AND** a feature after `tunnel` SHALL exit 2

## ADDED Requirements

### Requirement: Dirt is an app
`c3po dirt` SHALL run dirt with the remaining arguments. `c3po` and `c3po help` SHALL list `dirt` between `deport` and `gates`. `c3po help -f dirt` SHALL print the same text as `c3po dirt help`. A feature after `dirt` SHALL be an unknown feature and SHALL exit 2. There SHALL be no dirt screen. `c3po tui dirt` SHALL exit 2. The usage feature SHALL be `dirt`; help and version SHALL stay `help` and `version`. The usage record SHALL NOT store the scan path.

#### Scenario: Dirt is listed with the apps
- **WHEN** the operator runs `c3po`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `dirt`

#### Scenario: Dirt help is the category help
- **WHEN** the operator runs `c3po help -f dirt`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL match `c3po dirt help`
- **AND** `c3po help -f dirt extra` SHALL exit 2

#### Scenario: Dirt usage feature
- **WHEN** the agent runs `c3po dirt /tmp/work`
- **THEN** the recorded feature SHALL be `dirt`
- **AND** `c3po dirt help` SHALL be recorded as `help`
