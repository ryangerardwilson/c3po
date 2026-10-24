# xl Specification

## Purpose
TBD - created by archiving change xl-tui. Update Purpose after archive.
## Requirements
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

### Requirement: dd deletes the current row
The first `d` in normal mode SHALL arm deletion and SHALL show `d`. A second `d` SHALL delete the current row, SHALL move later rows up, and SHALL reload their formulas. A reference to the deleted row SHALL show an error. A range that contained the deleted row SHALL shrink. Esc SHALL cancel the arm. Any other key SHALL clear the arm and SHALL run as a normal motion. `d` typed while editing a cell SHALL NOT delete a row.

#### Scenario: dd deletes the current row
- **WHEN** the operator presses `dd` on a row that has a value
- **THEN** that value SHALL be gone
- **AND** the row below SHALL move up

#### Scenario: A range shrinks
- **WHEN** a formula sums a range and the operator deletes a row inside that range
- **THEN** the formula SHALL sum the remaining rows

#### Scenario: Another key cancels the arm
- **WHEN** the operator presses `d` and then `j`
- **THEN** the current row SHALL remain
- **AND** the cursor SHALL move

#### Scenario: Esc cancels the arm
- **WHEN** the operator presses `d` and then Esc
- **THEN** the current row SHALL remain

#### Scenario: Editing types d
- **WHEN** the operator is editing a cell and types `d`
- **THEN** the cell text SHALL contain `d`
- **AND** the row SHALL remain

### Requirement: Commands save and quit
`:w` SHALL save the workbook. `:wq` SHALL save the workbook and SHALL quit. `:q!` SHALL quit and SHALL discard unsaved changes. `q`, ctrl-c, and `:q` SHALL NOT quit. The screen SHALL NOT ask whether to save or discard.

#### Scenario: :wq saves and quits
- **WHEN** the operator runs `:wq` on a modified workbook that has a path
- **THEN** the file SHALL contain the modification
- **AND** the screen SHALL quit

#### Scenario: :q! discards
- **WHEN** the operator runs `:q!` on a modified workbook
- **THEN** the screen SHALL quit
- **AND** the file SHALL NOT gain that modification

#### Scenario: q stays
- **WHEN** the operator presses `q` on a modified workbook
- **THEN** the screen SHALL stay open
- **AND** the modification SHALL remain

