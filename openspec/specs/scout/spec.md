# scout

## Purpose

c3po selects a scouting tool and runs it.

## Requirements

### Requirement: Scout selects a tool
`c3po scout ast` SHALL run the ast tool with the remaining arguments. `c3po scout vibetree` SHALL run the size tree with the remaining arguments. `c3po scout cyclotree` SHALL run the call tree with the remaining arguments. `c3po scout` with no tool SHALL list `ast`, `vibetree`, and `cyclotree` and SHALL exit 0. An unknown scout name SHALL exit 2 and SHALL list the scouts. The names `complexity`, `hide`, and `visualize` SHALL be unknown scouts.

#### Scenario: Scout lists the tools
- **WHEN** the operator runs `c3po scout`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `ast`, `vibetree`, and `cyclotree`
- **AND** the text SHALL NOT list `complexity` or `hide`

#### Scenario: Ast help is reached through scout
- **GIVEN** the c3po binary
- **WHEN** the operator runs `c3po scout ast help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name the patch command

#### Scenario: Vibetree help is reached through scout
- **WHEN** the operator runs `c3po scout vibetree help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name `-p`, `-m`, and `--json`

#### Scenario: Cyclotree help is reached through scout
- **WHEN** the operator runs `c3po scout cyclotree help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL describe `j`, `k`, `h`, and `l`

#### Scenario: A retired scout name is rejected
- **WHEN** the operator runs `c3po scout complexity`
- **THEN** the command SHALL exit 2
- **AND** the same rejection SHALL apply to `hide` and `visualize`

#### Scenario: An unknown scout is rejected
- **WHEN** the operator runs `c3po scout nope`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL list the known scouts, `vibetree` among them

### Requirement: The previous name selects ast
Invoking the binary as `rgw-ast` SHALL run the ast scout. The arguments after the program name SHALL be the scout arguments.

#### Scenario: rgw-ast help still describes patch
- **WHEN** the operator runs `rgw-ast help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name the patch command
