# scout

## Purpose

c3po selects a scouting tool and runs it.

## Requirements

### Requirement: Scout selects a tool
`c3po scout ast` SHALL run the ast tool with the remaining arguments. `c3po scout complexity` SHALL run the complexity tool with the remaining arguments. `c3po scout` with no tool SHALL list `ast` and `complexity` and SHALL exit 0. An unknown scout name SHALL exit 2 and SHALL list the scouts.

#### Scenario: Ast help is reached through scout
- **GIVEN** the c3po binary
- **WHEN** the operator runs `c3po scout ast help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name the patch command

#### Scenario: Complexity help is reached through scout
- **GIVEN** the c3po binary
- **WHEN** the operator runs `c3po scout complexity help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name the note command

#### Scenario: An unknown scout is rejected
- **WHEN** the operator runs `c3po scout nope`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL list the known scouts

### Requirement: The previous names select a scout
Invoking the binary as `rgw-ast` SHALL run the ast scout. Invoking it as `cx` or `complexity` SHALL run the complexity scout. The arguments after the program name SHALL be the scout arguments.

#### Scenario: rgw-ast help still describes patch
- **WHEN** the operator runs `rgw-ast help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name the patch command
