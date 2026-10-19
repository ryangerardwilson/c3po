# apps

## Purpose

c3po runs one app from the first argument.

## Requirements

### Requirement: The first argument selects an app
`c3po ast` SHALL run the ast tool with the remaining arguments. `c3po cyclotree` SHALL run the call tree with the remaining arguments. `c3po deport`, `c3po gcal`, `c3po gdrive`, `c3po gmail`, `c3po l`, `c3po o`, `c3po pass`, and `c3po tunnel` SHALL run those tools with the remaining arguments. `c3po` with no tool SHALL list `ast`, `cyclotree`, `deport`, `gcal`, `gdrive`, `gmail`, `l`, `o`, `pass`, and `tunnel`, and SHALL exit 0. An unknown command SHALL exit 2 and SHALL list the apps. The names `complexity`, `hide`, `visualize`, `vibetree`, and `scout` SHALL be unknown commands.

#### Scenario: c3po lists the apps
- **WHEN** the operator runs `c3po`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `ast`, `cyclotree`, `deport`, `gcal`, `gdrive`, `gmail`, `l`, `o`, `pass`, and `tunnel`
- **AND** the text SHALL NOT list `complexity`, `hide`, or `vibetree`

#### Scenario: Ast help is reached directly
- **GIVEN** the c3po binary
- **WHEN** the operator runs `c3po ast help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name the patch command

#### Scenario: Cyclotree help is reached directly
- **WHEN** the operator runs `c3po cyclotree help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL describe `j`, `k`, `h`, and `l`

#### Scenario: A retired name is rejected
- **WHEN** the operator runs `c3po complexity`
- **THEN** the command SHALL exit 2
- **AND** the same rejection SHALL apply to `hide`, `visualize`, `vibetree`, and `scout`

#### Scenario: An unknown command is rejected
- **WHEN** the operator runs `c3po nope`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL list the known apps, `cyclotree` among them

### Requirement: The previous name selects ast
Invoking the binary as `rgw-ast` SHALL run ast. The arguments after the program name SHALL be the ast arguments.

#### Scenario: rgw-ast help still describes patch
- **WHEN** the operator runs `rgw-ast help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name the patch command

### Requirement: One help command
`c3po help` SHALL exit 0, SHALL show `c3po help -f`, and SHALL list `ast`, `cyclotree`, `deport`, `gcal`, `gdrive`, `gmail`, `l`, `o`, `pass`, and `tunnel`. `c3po help -f <category>` SHALL explain that category. `c3po help -f ast <feature>` SHALL explain that ast command. `c3po help -f cyclotree` SHALL describe `j` and `k`. `c3po help -f <app>` for each other app SHALL print the same text as `c3po <app> help`. An unknown category SHALL exit 2. An unknown ast feature SHALL exit 2. A feature after a non-ast app SHALL exit 2. `c3po help -f vibetree` SHALL exit 2.

#### Scenario: Help shows the single form
- **WHEN** the operator runs `c3po help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL show `c3po help -f`
- **AND** the text SHALL list `ast`, `cyclotree`, `deport`, `gcal`, `gdrive`, `gmail`, `l`, `o`, `pass`, and `tunnel`
- **AND** the text SHALL NOT list `vibetree`

#### Scenario: A category explains that tool
- **WHEN** the operator runs `c3po help -f cyclotree`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL describe `j` and `k`

#### Scenario: A feature explains one ast command
- **WHEN** the operator runs `c3po help -f ast patch`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL describe patch

#### Scenario: An unknown category is rejected
- **WHEN** the operator runs `c3po help -f nope`
- **THEN** the command SHALL exit 2
- **AND** `c3po help -f vibetree` SHALL exit 2

#### Scenario: An unknown feature is rejected
- **WHEN** the operator runs `c3po help -f ast nope`
- **THEN** the command SHALL exit 2
- **AND** a feature after `cyclotree` SHALL exit 2

### Requirement: Config lives under c3po
Each app's config SHALL be `$XDG_CONFIG_HOME/c3po/<app>` when `XDG_CONFIG_HOME` is set, and `~/.config/c3po/<app>` otherwise. Data SHALL be `$XDG_DATA_HOME/c3po/<app>` or `~/.local/share/c3po/<app>`. Cache SHALL be `$XDG_CACHE_HOME/c3po/<app>` or `~/.cache/c3po/<app>`. The pass token SHALL be the file `service-account-token` in the pass config directory. The directory `~/.config/op` SHALL stay in place. `GMAIL_CLI_CONFIG`, `GDRIVE_CONFIG`, `OP_SERVICE_ACCOUNT_TOKEN`, and `OP_SERVICE_ACCOUNT_TOKEN_FILE` SHALL still override the file they name.

#### Scenario: A legacy directory is adopted
- **GIVEN** config exists at the legacy path and the c3po path is absent
- **WHEN** c3po runs
- **THEN** the legacy path SHALL be renamed under `c3po/<app>`
- **AND** an existing c3po path SHALL be left as it is
