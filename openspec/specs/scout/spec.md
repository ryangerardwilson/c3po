# apps

## Purpose

c3po runs one app from the first argument. There is no tui command. Screens open through r2d2.
## Requirements
### Requirement: The first argument selects an app
`c3po ast` SHALL run the ast tool with the remaining arguments. `c3po clippy`, `c3po deport`, `c3po gcal`, `c3po gdrive`, `c3po gmail`, `c3po idealake`, `c3po looptab`, `c3po pass`, `c3po slack`, and `c3po tunnel` SHALL run those tools with the remaining arguments. There is no `c3po tui` command. `c3po tui` SHALL exit 2. `c3po` SHALL NOT open a terminal screen. Agents use the c3po commands and do not open a screen. `r2d2` is the terminal program, and running it opens the screen menu. The screens in this module SHALL be `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, and `looptab`, and `r2d2 <screen>` SHALL open them, including `r2d2 <screen> --resume <id>`. `c3po` with no tool SHALL list `ast`, `clippy`, `deport`, `gcal`, `gdrive`, `gmail`, `idealake`, `looptab`, `pass`, `slack`, and `tunnel`, and SHALL exit 0. It SHALL NOT list `tui`. An unknown command SHALL exit 2 and SHALL list the apps. The names `complexity`, `hide`, `visualize`, `vibetree`, `scout`, `cyclotree`, `l`, `o`, `xl`, `nlp`, and `tui` SHALL be unknown commands. `c3po tunnel open` SHALL exit 2. `c3po tui <screen>` and `c3po tui <screen> --resume <id>` SHALL exit 2.

#### Scenario: c3po lists the apps
- **WHEN** the operator runs `c3po`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `ast`, `clippy`, `deport`, `gcal`, `gdrive`, `gmail`, `idealake`, `looptab`, `pass`, `slack`, and `tunnel`
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
- **AND** the text SHALL say in vim

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

### Requirement: The previous name selects ast
Invoking the binary as `rgw-ast` SHALL run ast. The arguments after the program name SHALL be the ast arguments.

#### Scenario: rgw-ast help still describes patch
- **WHEN** the operator runs `rgw-ast help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name the patch command

### Requirement: One help command
`c3po help` SHALL exit 0, SHALL show `c3po help -f`, and SHALL list `ast`, `clippy`, `deport`, `gcal`, `gdrive`, `gmail`, `idealake`, `looptab`, `pass`, `slack`, and `tunnel`. It SHALL NOT list `tui`. `c3po help -f <category>` SHALL explain that category. `c3po help -f ast <feature>` SHALL explain that ast command. `c3po help -f <app>` SHALL print the same text as `c3po <app> help`. An unknown category SHALL exit 2. An unknown ast feature SHALL exit 2. A feature after an app other than ast SHALL exit 2. `c3po help -f tui`, `c3po help -f tui cyclotree`, `c3po help -f vibetree`, `c3po help -f cyclotree`, `c3po help -f l`, `c3po help -f o`, and `c3po help -f nlp` SHALL exit 2.

#### Scenario: Help shows the single form
- **WHEN** the operator runs `c3po help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL show `c3po help -f`
- **AND** the text SHALL list `ast`, `clippy`, `deport`, `gcal`, `gdrive`, `gmail`, `idealake`, `looptab`, `pass`, `slack`, and `tunnel`
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

### Requirement: Feature usage is recorded
Every c3po invocation SHALL add one to a daily counter for one catalog feature and one actor. The command SHALL only append the increment to a local journal under a file lock and return; it SHALL NOT run sql or dial Postgres. A detached worker SHALL write the journal to the runtime sqlite and then publish that sqlite with nebula s3 put to s3://sqlx/c3po.sqlite, and SHALL keep running until no increment has arrived for a short idle period, so increments that arrive while it works are drained by the same worker rather than one worker per command. A missing runtime copy SHALL keep the increment in the journal and start a background restore. With `SQLX_ROOT` set (and `SQLX_ENGINE` not `postgres`) or with `SQLX_ENGINE=sqlite`, the increment SHALL be written directly to that sqlite. The report SHALL include those increments. The actor SHALL be `human`, `agent`, or `service`. `C3PO_ACTOR` SHALL select the actor when its value is exactly one of those three words. The counter SHALL NOT store arguments, paths, presets, ports, or message text. Help and version SHALL be recorded as `help` and `version`, including `c3po help -f ast patch` and `c3po ast patch --help`. A failure to write the counter SHALL NOT change the command's exit code and SHALL NOT print on stderr. `c3po help` SHALL name `c3po usage`. `c3po usage` SHALL exit 0 and SHALL list every catalog feature that has no recorded use, then the features that have been used, with separate human, agent, and service counts, the number of exits 0, and the UTC date of the last use. A feature that was used and never exited 0 SHALL appear under used. The report SHALL be written before that same invocation is added to the counter. A stored feature that is not in the catalog SHALL be listed after the used features.

#### Scenario: An unused feature is listed
- **WHEN** the operator runs `c3po usage` and `ast cut` has never been recorded
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `ast cut` under `unused`

#### Scenario: Human and agent counts stay separate
- **GIVEN** one successful run recorded as a human and one successful run of the same feature recorded as an agent
- **WHEN** the operator runs `c3po usage`
- **THEN** that feature's row SHALL show both counts

#### Scenario: Help does not count as the feature
- **WHEN** the operator runs `c3po help -f ast patch`
- **THEN** the recorded feature SHALL be `help`
- **AND** `ast patch` SHALL remain unused until some invocation records `ast patch`

#### Scenario: A recording failure leaves the exit code alone
- **WHEN** the usage record cannot be written during `c3po version`
- **THEN** the command SHALL exit 0
- **AND** stderr SHALL NOT mention the usage record

### Requirement: Idealake is an app
`c3po idealake` SHALL run idealake with the remaining arguments. `c3po` and `c3po help` SHALL list `idealake`. `c3po help -f idealake` SHALL print the same text as `c3po idealake help`. `c3po tui idealake` SHALL exit 2.

#### Scenario: Idealake is listed with the apps
- **WHEN** the operator runs `c3po`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `idealake`

#### Scenario: Idealake help is the category help
- **WHEN** the operator runs `c3po help -f idealake`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL match `c3po idealake help`

#### Scenario: Idealake is not a screen
- **WHEN** the operator runs `c3po tui idealake`
- **THEN** the command SHALL exit 2

