## ADDED Requirements

### Requirement: The navigator draws a block cursor
The command line and the filter SHALL draw the cursor as one reverse-video cell. When the cursor is on a character, that cell SHALL be the character. When the cursor is at the end of the text, that cell SHALL be a space. The drawn cursor SHALL NOT be a vertical bar. While the navigator owns the terminal, the terminal cursor SHALL be a steady block. When the navigator gives the terminal back, including for a shell command, the terminal cursor SHALL return to the terminal default. The listing cursor marker SHALL stay `>`.

#### Scenario: The colon cursor is a block
- **WHEN** the operator presses `:`
- **THEN** the last line SHALL show `:` followed by a reverse-video block
- **AND** the last line SHALL NOT contain a vertical bar

#### Scenario: The cursor covers the current character
- **WHEN** the operator presses `:`, types `mkdir notes`, and presses Ctrl-a
- **THEN** the reverse-video cell SHALL be `m`

#### Scenario: The filter cursor is a block
- **WHEN** the operator presses `/`
- **THEN** the filter line SHALL show `/` followed by a reverse-video block
- **AND** the filter line SHALL NOT contain a vertical bar
