# scout

## Purpose

c3po selects a scouting tool and runs it.

## Requirements

### Requirement: Scout selects a tool
`c3po scout ast` SHALL run the ast tool with the remaining arguments. `c3po scout cyclotree` SHALL run the call tree with the remaining arguments. `c3po scout` with no tool SHALL list `ast` and `cyclotree` and SHALL exit 0. An unknown scout name SHALL exit 2 and SHALL list the scouts. The names `complexity`, `hide`, `visualize`, and `vibetree` SHALL be unknown scouts.

#### Scenario: Scout lists the tools
- **WHEN** the operator runs `c3po scout`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `ast` and `cyclotree`
- **AND** the text SHALL NOT list `complexity`, `hide`, or `vibetree`

#### Scenario: Ast help is reached through scout
- **GIVEN** the c3po binary
- **WHEN** the operator runs `c3po scout ast help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name the patch command

#### Scenario: Cyclotree help is reached through scout
- **WHEN** the operator runs `c3po scout cyclotree help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL describe `j`, `k`, `h`, and `l`

#### Scenario: A retired scout name is rejected
- **WHEN** the operator runs `c3po scout complexity`
- **THEN** the command SHALL exit 2
- **AND** the same rejection SHALL apply to `hide`, `visualize`, and `vibetree`

#### Scenario: An unknown scout is rejected
- **WHEN** the operator runs `c3po scout nope`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL list the known scouts, `cyclotree` among them

### Requirement: The previous name selects ast
Invoking the binary as `rgw-ast` SHALL run the ast scout. The arguments after the program name SHALL be the scout arguments.

#### Scenario: rgw-ast help still describes patch
- **WHEN** the operator runs `rgw-ast help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name the patch command

### Requirement: One help command
`c3po help` SHALL exit 0, SHALL show `c3po help -f`, and SHALL list `ast` and `cyclotree`. `c3po help -f <category>` SHALL explain that category. `c3po help -f ast <feature>` SHALL explain that ast command. `c3po help -f cyclotree` SHALL describe `j` and `k`. An unknown category SHALL exit 2. An unknown ast feature SHALL exit 2. A feature after `cyclotree` SHALL exit 2. `c3po help -f vibetree` SHALL exit 2.

#### Scenario: Help shows the single form
- **WHEN** the operator runs `c3po help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL show `c3po help -f`
- **AND** the text SHALL list `ast` and `cyclotree`
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
