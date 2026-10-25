## REMOVED Requirements

### Requirement: Bang runs touch, mkdir, or rm
**Reason**: `:` owns the touch, mkdir, and rm command line. `!` runs a shell command.
**Migration**: Press `:` for touch, mkdir, or rm. Press `!` for any shell command in the listed directory.

## ADDED Requirements

### Requirement: Colon runs touch, mkdir, or rm
`:` SHALL open a command line on the last row. The command line SHALL show `:` followed by the text being typed, and SHALL NOT show that text on the prompt line under the title. Enter SHALL run that line in the listed directory when the command is `touch`, `mkdir`, or `rm`. `mkdir` MAY take `-p`. Any other command or flag SHALL be refused and SHALL NOT run. Esc SHALL cancel the command line without running a command. After a command runs, the listing SHALL reload and the status line SHALL be the last line again. `:` typed into the filter SHALL stay text. The shortcut list and `c3po tui o help` SHALL name `:`.

#### Scenario: mkdir creates a directory
- **WHEN** the operator presses `:`, types `mkdir notes`, and presses enter
- **THEN** the listed directory SHALL contain a directory named notes
- **AND** the listing SHALL show notes

#### Scenario: Another command is refused
- **WHEN** the operator submits `rm -rf notes` on the `:` command line
- **THEN** no command SHALL run
- **AND** the directory notes SHALL remain

#### Scenario: The command line is the last row
- **WHEN** the operator presses `:` and types `mkdir notes`
- **THEN** the last line SHALL contain `mkdir notes`
- **AND** the line under the title SHALL NOT contain `mkdir notes`

#### Scenario: Colon in the filter is text
- **WHEN** the operator is typing a filter and presses `:`
- **THEN** the command line SHALL stay closed
- **AND** the filter text SHALL contain `:`

### Requirement: Bang runs a shell command
`!` SHALL open a command line on the last row. The command line SHALL show `!` followed by the text being typed, and SHALL NOT show that text on the prompt line under the title. Enter SHALL run that line in the shell with the listed directory as its working directory, and SHALL hand the terminal to that command. The command SHALL NOT be limited to `touch`, `mkdir`, or `rm`. An empty line SHALL NOT run. Esc SHALL cancel the command line without running a command. After the command exits, the listing SHALL reload and the status line SHALL be the last line again. `!` typed into the filter SHALL stay text. The shortcut list and `c3po tui o help` SHALL name `!`.

#### Scenario: A shell command runs in the listed directory
- **WHEN** the operator presses `!`, types `touch made.txt`, and presses enter
- **THEN** `made.txt` SHALL be created in the listed directory

#### Scenario: A refused file command is allowed in the shell
- **WHEN** the operator presses `!`, types `rm -rf notes`, and presses enter
- **THEN** the directory notes SHALL be gone

#### Scenario: An empty shell line does not run
- **WHEN** the operator presses `!` and then enter
- **THEN** no command SHALL run

#### Scenario: The shell line is the last row
- **WHEN** the operator presses `!` and types `touch made.txt`
- **THEN** the last line SHALL contain `touch made.txt`
- **AND** the line under the title SHALL NOT contain `touch made.txt`

## MODIFIED Requirements

### Requirement: Visual range
`v` SHALL toggle visual mode. The first `v` on a row SHALL anchor the range on that row and the status SHALL be `visual 1`. Motion keys SHALL move the cursor and SHALL extend the range through every listing row between the anchor and the cursor. The range SHALL stay inside the listing and SHALL NOT wrap. Each row in the range SHALL be highlighted, and the cursor row SHALL keep its cursor marker. A second `v` SHALL leave visual mode and SHALL leave every file in place. Esc SHALL leave visual mode and SHALL leave every file in place. `h` SHALL leave visual mode. The Delete key and Backspace SHALL delete every file and directory in the range, SHALL leave visual mode, and SHALL NOT delete when visual mode is off. `x` SHALL stage the range to move, in listing order, and SHALL leave visual mode. `y` SHALL stage the range to copy, in listing order, and SHALL leave visual mode. `v`, the Delete key, or Backspace typed into the filter or a command prompt SHALL NOT enter visual mode and SHALL NOT delete a file. Backspace in the filter or a command prompt SHALL edit that text. An empty listing SHALL leave visual mode off. The shortcut list and `c3po tui o help` SHALL name `v`, Backspace, and the Delete key.

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

#### Scenario: Backspace removes the range
- **WHEN** the operator selects two files with `v` and `j` and presses Backspace
- **THEN** both files SHALL be gone
- **AND** visual mode SHALL be off

#### Scenario: Backspace does nothing outside visual mode
- **WHEN** visual mode is off and the operator presses Backspace
- **THEN** the current file SHALL remain

#### Scenario: Backspace in the filter edits text
- **WHEN** the operator is typing a filter and presses Backspace
- **THEN** the filter text SHALL drop its last character
- **AND** the current file SHALL remain

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
