# gates Specification

## Purpose
The gates command is gone. The rules that used to be printed here live as r2d2 skills. c3po does not print a statute and does not tally a sheet.
## Requirements
### Requirement: Gates is not a command
`c3po gates` SHALL be an unknown command and SHALL exit 2. `c3po gates --help` and `c3po gates version` SHALL also be unknown commands and SHALL exit 2. `c3po` and `c3po help` SHALL NOT list `gates`. The usage feature SHALL be `unknown`. There SHALL be no gates screen.

#### Scenario: The name is rejected
- **WHEN** the operator runs `c3po gates`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL say unknown command

#### Scenario: Help and version of that name are rejected
- **WHEN** the operator runs `c3po gates --help`
- **THEN** the command SHALL exit 2
- **AND** `c3po gates version` SHALL exit 2

#### Scenario: The apps list omits gates
- **WHEN** the operator runs `c3po`
- **THEN** the text SHALL NOT list `gates`

