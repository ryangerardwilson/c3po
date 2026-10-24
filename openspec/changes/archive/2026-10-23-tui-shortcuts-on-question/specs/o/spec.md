## MODIFIED Requirements

### Requirement: The screen fills the terminal
The navigator view SHALL contain one line for every row of the terminal. The title SHALL be the first line and the status line SHALL be the last line. The listing SHALL use the rows between the prompt line and the status line. A listing with fewer entries SHALL pad those rows so the status line stays on the last line. The listing SHALL NOT show a shortcut legend.

#### Scenario: A short directory fills the screen
- **WHEN** the terminal is 24 rows
- **AND** the listed directory has one entry
- **THEN** the view SHALL be 24 lines
- **AND** the last line SHALL be the status line
- **AND** the view SHALL NOT contain `j/k move`
