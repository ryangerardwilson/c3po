# o Specification

## Purpose
TBD - created by archiving change o-direct-path. Update Purpose after archive.
## Requirements
### Requirement: A path is given directly
`c3po tui o <path> [<path> ...]` SHALL open those paths. One directory SHALL start the navigator at that directory. One or more files SHALL open in `$VISUAL`, `$EDITOR`, or vim when every file is text, and SHALL be handed to xdg-open otherwise, except a file whose name ends in `.xlsx`, which SHALL open in the xl screen in the foreground. The navigator SHALL open that same `.xlsx` file in the xl screen and SHALL return when the spreadsheet exits. Several targets SHALL all be files. A missing path SHALL exit 1 and SHALL say `Path not found`. The word `open` SHALL NOT be a command. `c3po tui o` with no arguments SHALL start the navigator in the working directory. `c3po tui o help` SHALL show a direct path example, SHALL say an xlsx file opens in xl, and SHALL NOT show `c3po tui o open`.

#### Scenario: A directory starts the navigator
- **WHEN** the operator runs `c3po tui o` with one directory
- **THEN** the navigator SHALL start at that directory

#### Scenario: A missing path is reported
- **WHEN** the operator runs `c3po tui o` with a path that does not exist
- **THEN** the command SHALL exit 1
- **AND** the text SHALL say `Path not found`

#### Scenario: The word open is a path
- **WHEN** the operator runs `c3po tui o open` and no path named open exists
- **THEN** the command SHALL exit 1
- **AND** the text SHALL say `Path not found`
- **AND** the command SHALL NOT print the usage line

#### Scenario: A workbook opens in xl
- **WHEN** the operator runs `c3po tui o` with one `.xlsx` file
- **THEN** that file SHALL open in the xl screen
- **AND** it SHALL NOT be handed to xdg-open

### Requirement: Named commands stay
`reveal`, `pick`, `save`, and `config` SHALL stay commands. `help` and `version` SHALL stay standalone. A word that is not one of those commands SHALL be read as a path.

#### Scenario: Reveal still takes one path
- **WHEN** the operator runs `c3po tui o reveal` with no path
- **THEN** the command SHALL exit 1
- **AND** the text SHALL say `Use: c3po tui o reveal <path>`

### Requirement: The screen fills the terminal
The navigator view SHALL contain one line for every row of the terminal. The title SHALL be the first line. While the command line is closed, the status line SHALL be the last line. While the command line is open, the last line SHALL be the command line. The listing SHALL use the rows between the prompt line and the last line. A listing with fewer entries SHALL pad those rows so the last line stays at the bottom. The listing SHALL NOT show a shortcut legend.

#### Scenario: A short directory fills the screen
- **WHEN** the terminal is 24 rows
- **AND** the listed directory has one entry
- **THEN** the view SHALL be 24 lines
- **AND** the last line SHALL be the status line
- **AND** the view SHALL NOT contain `j/k move`

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

### Requirement: Colon runs touch, mkdir, or rm
`:` SHALL open a command line on the last row. The command line SHALL show `:` followed by the text being typed, and SHALL NOT show that text on the prompt line under the title. Enter SHALL run that line in the listed directory when the command is `touch`, `mkdir`, or `rm`. `mkdir` MAY take `-p`. Any other command or flag SHALL be refused and SHALL NOT run. A line that starts with `!` SHALL run the rest in the shell with the listed directory as its working directory, and SHALL hand the terminal to that command. That command SHALL NOT be limited to `touch`, `mkdir`, or `rm`. Space around that shell command SHALL be ignored. An empty shell command SHALL NOT run. Esc SHALL cancel the command line without running a command. After a command runs, the listing SHALL reload and the status line SHALL be the last line again. `:` typed into the filter SHALL stay text. The shortcut list and `c3po tui o help` SHALL name `:` and `:!`.

#### Scenario: mkdir creates a directory
- **WHEN** the operator presses `:`, types `mkdir notes`, and presses enter
- **THEN** the listed directory SHALL contain a directory named notes
- **AND** the listing SHALL show notes

#### Scenario: Another command is refused
- **WHEN** the operator submits `rm -rf notes` on the `:` command line
- **THEN** no command SHALL run
- **AND** the directory notes SHALL remain

#### Scenario: Colon bang runs a shell command
- **WHEN** the operator presses `:`, types `!touch made.txt`, and presses enter
- **THEN** `made.txt` SHALL be created in the listed directory

#### Scenario: Colon bang allows a refused file command
- **WHEN** the operator presses `:`, types `!rm -rf notes`, and presses enter
- **THEN** the directory notes SHALL be gone

#### Scenario: An empty colon bang does not run
- **WHEN** the operator presses `:`, types `!`, and presses enter
- **THEN** no command SHALL run

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

