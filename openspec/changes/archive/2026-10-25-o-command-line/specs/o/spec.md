## MODIFIED Requirements

### Requirement: The screen fills the terminal
The navigator view SHALL contain one line for every row of the terminal. The title SHALL be the first line. While the command line is closed, the status line SHALL be the last line. While the command line is open, the last line SHALL be the command line. The listing SHALL use the rows between the prompt line and the last line. A listing with fewer entries SHALL pad those rows so the last line stays at the bottom. The listing SHALL NOT show a shortcut legend.

#### Scenario: A short directory fills the screen
- **WHEN** the terminal is 24 rows
- **AND** the listed directory has one entry
- **THEN** the view SHALL be 24 lines
- **AND** the last line SHALL be the status line
- **AND** the view SHALL NOT contain `j/k move`

### Requirement: Bang runs touch, mkdir, or rm
`!` SHALL open a command line on the last row. The command line SHALL show `!` followed by the text being typed, and SHALL NOT show that text on the prompt line under the title. Enter SHALL run that line in the listed directory when the command is `touch`, `mkdir`, or `rm`. `mkdir` MAY take `-p`. Any other command or flag SHALL be refused and SHALL NOT run. Esc SHALL cancel the command line without running a command. After a command runs, the listing SHALL reload and the status line SHALL be the last line again.

#### Scenario: mkdir creates a directory
- **WHEN** the operator presses `!`, types `mkdir notes`, and presses enter
- **THEN** the listed directory SHALL contain a directory named notes
- **AND** the listing SHALL show notes

#### Scenario: Another command is refused
- **WHEN** the operator submits `rm -rf notes`
- **THEN** no command SHALL run
- **AND** the directory notes SHALL remain

#### Scenario: The command line is the last row
- **WHEN** the operator presses `!` and types `mkdir notes`
- **THEN** the last line SHALL contain `mkdir notes`
- **AND** the line under the title SHALL NOT contain `mkdir notes`
