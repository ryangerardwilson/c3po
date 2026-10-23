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

