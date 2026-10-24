## ADDED Requirements

### Requirement: dd deletes the current item
The first `d` SHALL arm deletion and SHALL show `d`. A second `d` SHALL delete the current file or directory and SHALL reload the listing. A directory SHALL be removed with its contents. A symlink SHALL be removed without removing its target. Esc SHALL cancel the arm. Any other key SHALL clear the arm and SHALL run as a normal motion. An empty listing SHALL leave the directory unchanged and SHALL report that there is nothing to delete. `d` typed into the filter or the command prompt SHALL NOT delete.

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
