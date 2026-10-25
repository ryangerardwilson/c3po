## MODIFIED Requirements

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
