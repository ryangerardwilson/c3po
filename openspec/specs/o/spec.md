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
The navigator view SHALL contain one line for every row of the terminal. The title SHALL be the first line and the status line SHALL be the last line. The listing SHALL use the rows between the prompt line and the status line. A listing with fewer entries SHALL pad those rows so the status line stays on the last line. The listing SHALL NOT show a shortcut legend.

#### Scenario: A short directory fills the screen
- **WHEN** the terminal is 24 rows
- **AND** the listed directory has one entry
- **THEN** the view SHALL be 24 lines
- **AND** the last line SHALL be the status line
- **AND** the view SHALL NOT contain `j/k move`

### Requirement: Bang runs touch, mkdir, or rm
`!` SHALL open a command prompt. Enter SHALL run that line in the listed directory when the command is `touch`, `mkdir`, or `rm`. `mkdir` MAY take `-p`. Any other command or flag SHALL be refused and SHALL NOT run. Esc SHALL cancel the prompt without running a command. After a command runs, the listing SHALL reload.

#### Scenario: mkdir creates a directory
- **WHEN** the operator presses `!`, types `mkdir notes`, and presses enter
- **THEN** the listed directory SHALL contain a directory named notes
- **AND** the listing SHALL show notes

#### Scenario: Another command is refused
- **WHEN** the operator submits `rm -rf notes`
- **THEN** no command SHALL run
- **AND** the directory notes SHALL remain

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

