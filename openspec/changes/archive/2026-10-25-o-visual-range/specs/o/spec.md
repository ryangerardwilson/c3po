## MODIFIED Requirements

### Requirement: Cut, copy, and paste
When visual mode is off, `x` SHALL stage the current file or directory to move and `y` SHALL stage the current file or directory to copy. `p` SHALL place every staged item in the listed directory under its own name, in staged order. A copy SHALL leave each source in place and SHALL stay staged. A cut SHALL remove each source and SHALL clear the stage. `p` SHALL refuse when nothing is staged, when any destination name exists, when any staged path is missing, or when any paste would put a directory inside itself. A refused paste SHALL leave every source and every existing destination unchanged. `y` SHALL NOT send the path to `wl-copy`.

#### Scenario: A file is copied into another directory
- **WHEN** the operator presses `y` on a file and `p` in another directory
- **THEN** that directory SHALL contain the file
- **AND** the source file SHALL remain

#### Scenario: A file is cut into another directory
- **WHEN** the operator presses `x` on a file and `p` in another directory
- **THEN** that directory SHALL contain the file
- **AND** the source file SHALL be gone

#### Scenario: An existing name is refused
- **WHEN** the listed directory already contains the staged name
- **AND** the operator presses `p`
- **THEN** the existing file SHALL remain unchanged
- **AND** the source SHALL remain

#### Scenario: A directory is not pasted into itself
- **WHEN** the operator stages a directory and presses `p` inside that directory
- **THEN** the directory SHALL NOT contain a copy of itself

### Requirement: dd deletes the current item
When visual mode is off, the first `d` SHALL arm deletion and SHALL show `d`. A second `d` SHALL delete the current file or directory and SHALL reload the listing. A directory SHALL be removed with its contents. A symlink SHALL be removed without removing its target. Esc SHALL cancel the arm. Any other key SHALL clear the arm and SHALL run as a normal motion. An empty listing SHALL leave the directory unchanged and SHALL report that there is nothing to delete. `d` typed into the filter or the command prompt SHALL NOT delete. When visual mode is on, the second `d` SHALL delete every file and directory in the visual range and SHALL leave visual mode.

#### Scenario: dd deletes a file
- **WHEN** the operator presses `dd` on a file
- **THEN** that file SHALL be gone
- **AND** the listing SHALL no longer show it

#### Scenario: dd deletes a directory
- **WHEN** the operator presses `dd` on a directory that contains a file
- **THEN** that directory SHALL be gone

#### Scenario: Another key cancels the arm
- **WHEN** the operator presses `d` and then `j`
- **THEN** the current file SHALL remain
- **AND** the cursor SHALL move

#### Scenario: Esc cancels the arm
- **WHEN** the operator presses `d` and then Esc
- **THEN** the current file SHALL remain

#### Scenario: dd deletes the visual range
- **WHEN** visual mode covers two files
- **AND** the operator presses `dd`
- **THEN** both files SHALL be gone
- **AND** visual mode SHALL be off

## ADDED Requirements

### Requirement: Visual range
`v` SHALL toggle visual mode. The first `v` on a row SHALL anchor the range on that row and the status SHALL be `visual 1`. Motion keys SHALL move the cursor and SHALL extend the range through every listing row between the anchor and the cursor. The range SHALL stay inside the listing and SHALL NOT wrap. Each row in the range SHALL be highlighted, and the cursor row SHALL keep its cursor marker. A second `v` SHALL leave visual mode and SHALL leave every file in place. Esc SHALL leave visual mode and SHALL leave every file in place. `h` SHALL leave visual mode. The Delete key SHALL delete every file and directory in the range, SHALL leave visual mode, and SHALL NOT delete when visual mode is off. `x` SHALL stage the range to move, in listing order, and SHALL leave visual mode. `y` SHALL stage the range to copy, in listing order, and SHALL leave visual mode. `v` or the Delete key typed into the filter or the command prompt SHALL NOT enter visual mode and SHALL NOT delete. An empty listing SHALL leave visual mode off. The shortcut list and `c3po tui o help` SHALL name `v` and the Delete key.

#### Scenario: A range is highlighted
- **WHEN** the operator presses `v` on the first of three files and then `j` twice
- **THEN** those three files SHALL be highlighted
- **AND** the status SHALL be `visual 3`

#### Scenario: v again clears the range
- **WHEN** the operator presses `v` and then `v` again
- **THEN** visual mode SHALL be off
- **AND** the file SHALL remain

#### Scenario: The range does not wrap
- **WHEN** the operator presses `v` on the first file and then `k`
- **THEN** the selection SHALL stay on the first file
- **AND** the status SHALL be `visual 1`

#### Scenario: Delete removes the range
- **WHEN** the operator selects two files with `v` and `j` and presses Delete
- **THEN** both files SHALL be gone
- **AND** the other files SHALL remain
- **AND** visual mode SHALL be off

#### Scenario: Delete does nothing outside visual mode
- **WHEN** visual mode is off and the operator presses Delete
- **THEN** the current file SHALL remain

#### Scenario: y copies the range
- **WHEN** the operator selects two files and presses `y`, then `p` in another directory
- **THEN** that directory SHALL contain both files
- **AND** both sources SHALL remain
- **AND** the stage SHALL remain

#### Scenario: x cuts the range
- **WHEN** the operator selects two files and presses `x`, then `p` in another directory
- **THEN** that directory SHALL contain both files
- **AND** both sources SHALL be gone
- **AND** the stage SHALL be clear

#### Scenario: One existing name refuses the paste
- **WHEN** the operator has staged two names and the listed directory already contains one of them
- **AND** the operator presses `p`
- **THEN** no staged item SHALL be added
- **AND** every source SHALL remain

#### Scenario: v in the filter is text
- **WHEN** the operator is typing a filter and presses `v`
- **THEN** visual mode SHALL stay off
- **AND** the filter text SHALL contain `v`
