# apps

## Purpose

c3po runs one app from the first argument.
## Requirements
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

### Requirement: The previous name selects ast
Invoking the binary as `rgw-ast` SHALL run ast. The arguments after the program name SHALL be the ast arguments.

#### Scenario: rgw-ast help still describes patch
- **WHEN** the operator runs `rgw-ast help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name the patch command

### Requirement: One help command
`c3po help` SHALL exit 0, SHALL show `c3po help -f`, and SHALL list `ast`, `deport`, `gcal`, `gdrive`, `gmail`, `looptab`, `pass`, `slack`, `tunnel`, and `tui`. `c3po help -f <category>` SHALL explain that category. `c3po help -f ast <feature>` SHALL explain that ast command. `c3po help -f tui` SHALL list `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, and `looptab`. `c3po help -f tui <screen>` SHALL print the same text as `c3po tui <screen> help`. `c3po help -f tui cyclotree` SHALL describe `j` and `k`. `c3po help -f <app>` for each other app SHALL print the same text as `c3po <app> help`. An unknown category SHALL exit 2. An unknown ast feature SHALL exit 2. An unknown screen SHALL exit 2. A feature after an app other than ast or tui SHALL exit 2. `c3po help -f vibetree`, `c3po help -f cyclotree`, `c3po help -f l`, and `c3po help -f o` SHALL exit 2.

#### Scenario: Help shows the single form
- **WHEN** the operator runs `c3po help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL show `c3po help -f`
- **AND** the text SHALL list `ast`, `deport`, `gcal`, `gdrive`, `gmail`, `looptab`, `pass`, `slack`, `tunnel`, and `tui`
- **AND** the text SHALL NOT list `vibetree`

#### Scenario: A category explains that screen
- **WHEN** the operator runs `c3po help -f tui cyclotree`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL describe `j` and `k`

#### Scenario: Spreadsheet help is the screen help
- **WHEN** the operator runs `c3po help -f tui xl`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL match `c3po tui xl help`

#### Scenario: Looptab help is the screen help
- **WHEN** the operator runs `c3po help -f tui looptab`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL match `c3po tui looptab help`

#### Scenario: A feature explains one ast command
- **WHEN** the operator runs `c3po help -f ast patch`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL describe patch

#### Scenario: An unknown category is rejected
- **WHEN** the operator runs `c3po help -f nope`
- **THEN** the command SHALL exit 2
- **AND** `c3po help -f vibetree` SHALL exit 2
- **AND** `c3po help -f cyclotree`, `c3po help -f l`, and `c3po help -f o` SHALL exit 2

#### Scenario: An unknown feature is rejected
- **WHEN** the operator runs `c3po help -f ast nope`
- **THEN** the command SHALL exit 2
- **AND** `c3po help -f tui nope` SHALL exit 2
- **AND** a feature after `tunnel` SHALL exit 2

### Requirement: Config lives under c3po
Each app's config SHALL be `$XDG_CONFIG_HOME/c3po/<app>` when `XDG_CONFIG_HOME` is set, and `~/.config/c3po/<app>` otherwise. A data directory MAY remain at `$XDG_DATA_HOME/c3po/<app>` or `~/.local/share/c3po/<app>`. Cache SHALL be `$XDG_CACHE_HOME/c3po/<app>` or `~/.cache/c3po/<app>`. Durable records SHALL live in the sqlx database `c3po`: OAuth tokens for gmail, gcal, and gdrive, gdrive sync state, and tunnel routes. Operator-edited config, credential files, signatures, the pass service-account token, generated tunnel credentials, and caches SHALL stay files. The pass token SHALL be the file `service-account-token` in the pass config directory. The directory `~/.config/op` SHALL stay in place. `GMAIL_CLI_CONFIG`, `GDRIVE_CONFIG`, `OP_SERVICE_ACCOUNT_TOKEN`, and `OP_SERVICE_ACCOUNT_TOKEN_FILE` SHALL still override the file they name.

#### Scenario: A legacy directory is adopted
- **GIVEN** config exists at the legacy path and the c3po path is absent
- **WHEN** c3po runs
- **THEN** the legacy path SHALL be renamed under `c3po/<app>`
- **AND** an existing c3po path SHALL be left as it is

#### Scenario: Durable records use one database
- **WHEN** c3po saves an OAuth token, Drive sync state, or a tunnel route
- **THEN** the record SHALL be stored in the sqlx database `c3po`
- **AND** a config file, credential file, signature, or cache SHALL stay a file

### Requirement: The program name xl does not select the spreadsheet
Invoking the binary as `xl` SHALL NOT run the spreadsheet. `xl version` SHALL NOT print `0.1.0`. The installer SHALL NOT create an `xl` link. The installer SHALL remove an `xl` symlink that points at the c3po binary.

#### Scenario: The program name is not the spreadsheet
- **WHEN** the operator runs the binary as `xl version`
- **THEN** the text SHALL NOT be `0.1.0`
- **AND** `xl help` SHALL NOT describe the budget grid

