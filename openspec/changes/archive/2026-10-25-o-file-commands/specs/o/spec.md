## REMOVED Requirements

### Requirement: Colon runs touch, mkdir, or rm
**Reason**: The colon line runs mv, rm, cp, touch, and mkdir. rm may take -rf. It shows no hint and does not run a shell.
**Migration**: Type one of those commands after `:`.

## ADDED Requirements

### Requirement: Colon runs mv, rm, cp, touch, or mkdir
`:` SHALL open a command line on the last row. The command line SHALL show `:` followed by the text being typed, and SHALL NOT show a hint or that text on the prompt line under the title. Enter SHALL run that line in the listed directory when the command is `mv`, `rm`, `cp`, `touch`, or `mkdir`. `rm` MAY take `-rf`. `cp` and `mv` SHALL require two paths. Any other command or flag SHALL be refused and SHALL NOT run. A line that starts with `!` SHALL NOT run a shell command. Esc SHALL cancel the command line without running a command. After a command runs, the listing SHALL reload and the status line SHALL be the last line again. `:` typed into the filter SHALL stay text. The shortcut list and `c3po tui o help` SHALL name these commands.

#### Scenario: mkdir creates a directory
- **WHEN** the operator presses `:`, types `mkdir notes`, and presses enter
- **THEN** the listed directory SHALL contain a directory named notes
- **AND** the listing SHALL show notes

#### Scenario: cp copies a file
- **WHEN** the operator submits `cp a.txt b.txt` on the `:` command line
- **THEN** the listed directory SHALL contain b.txt
- **AND** a.txt SHALL remain

#### Scenario: mv renames a file
- **WHEN** the operator submits `mv b.txt c.txt` on the `:` command line
- **THEN** the listed directory SHALL contain c.txt
- **AND** b.txt SHALL be gone

#### Scenario: rm -rf removes a directory
- **WHEN** the operator submits `rm -rf notes` on the `:` command line
- **THEN** the directory notes SHALL be gone

#### Scenario: Another command is refused
- **WHEN** the operator submits `ls` on the `:` command line
- **THEN** no command SHALL run

#### Scenario: Another flag is refused
- **WHEN** the operator submits `rm -r notes` on the `:` command line
- **THEN** no command SHALL run
- **AND** the directory notes SHALL remain

#### Scenario: A colon bang does not run
- **WHEN** the operator submits `!touch made.txt` on the `:` command line
- **THEN** no command SHALL run
- **AND** made.txt SHALL NOT be created

#### Scenario: The command line is the last row
- **WHEN** the operator presses `:` and types `mkdir notes`
- **THEN** the last line SHALL contain `mkdir notes`
- **AND** the line under the title SHALL NOT contain `mkdir notes`

#### Scenario: An empty colon line has no hint
- **WHEN** the operator presses `:`
- **THEN** the last line SHALL contain `:`
- **AND** the last line SHALL NOT name touch, mkdir, or rm

#### Scenario: Colon in the filter is text
- **WHEN** the operator is typing a filter and presses `:`
- **THEN** the command line SHALL stay closed
- **AND** the filter text SHALL contain `:`
