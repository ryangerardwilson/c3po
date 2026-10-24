## ADDED Requirements

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
