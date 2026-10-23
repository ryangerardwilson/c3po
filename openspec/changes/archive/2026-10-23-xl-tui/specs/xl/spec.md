## ADDED Requirements

### Requirement: The spreadsheet is a screen
`c3po tui xl` SHALL run the spreadsheet. With no arguments, when stdout is not a terminal, the command SHALL print help and SHALL exit 0. `c3po tui xl help` SHALL exit 0 and SHALL name an xlsx file and `c3po tui xl`. `c3po tui xl version` SHALL print `0.1.0`. One path argument SHALL open that workbook when stdout is a terminal. Two path arguments SHALL exit 1. `c3po xl` SHALL exit 2.

#### Scenario: Help names the workbook
- **WHEN** the operator runs `c3po tui xl help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name an xlsx file
- **AND** the text SHALL name `c3po tui xl`

#### Scenario: Version is the spreadsheet version
- **WHEN** the operator runs `c3po tui xl version`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL be `0.1.0`

#### Scenario: Two paths are rejected
- **WHEN** the operator runs `c3po tui xl` with two paths
- **THEN** the command SHALL exit 1

#### Scenario: A bare name is rejected
- **WHEN** the operator runs `c3po xl`
- **THEN** the command SHALL exit 2
