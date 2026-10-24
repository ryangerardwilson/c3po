## ADDED Requirements

### Requirement: Cut, copy, and paste
`x` SHALL stage the current file or directory to move. `y` SHALL stage the current file or directory to copy. `p` SHALL place the staged item in the listed directory under the same name. A copy SHALL leave the source in place and SHALL stay staged. A cut SHALL remove the source and SHALL clear the stage. `p` SHALL refuse when nothing is staged, when the destination name exists, or when the paste would put a directory inside itself. `y` SHALL NOT send the path to `wl-copy`.

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
