## ADDED Requirements

### Requirement: The screen fills the terminal
The navigator view SHALL contain one line for every row of the terminal. The title SHALL be the first line and the key hint SHALL be the last line. The listing SHALL use the rows between the prompt line and the status line. A listing with fewer entries SHALL pad those rows so the key hint stays on the last line.

#### Scenario: A short directory fills the screen
- **WHEN** the terminal is 24 rows
- **AND** the listed directory has one entry
- **THEN** the view SHALL be 24 lines
- **AND** the last line SHALL be the key hint

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
