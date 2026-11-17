# clippy Specification

## Purpose
TBD - created by archiving change add-clippy. Update Purpose after archive.
## Requirements
### Requirement: Clippy copies a multiline text
`c3po clippy` with no arguments SHALL read stdin until EOF and copy those bytes to the Wayland clipboard through `wl-copy`. The bytes SHALL be the stdin of `wl-copy` and SHALL NOT be a `wl-copy` argument. `c3po clippy <text...>` SHALL copy the arguments joined by one space. A newline inside an argument SHALL be kept. The command SHALL NOT print the text on stdout or stderr. On success stdout SHALL be exactly `copied` and the process SHALL exit 0.

An empty text SHALL NOT be copied. Stderr SHALL name the usage and the process SHALL exit 2. When `wl-copy` is missing, the process SHALL exit 1 and stderr SHALL be `wl-copy is required.` When `wl-copy` fails, the process SHALL exit 1 and stderr SHALL be `Unable to copy to the clipboard.` A read error SHALL exit 1 and SHALL NOT print the text.

When stdin is a terminal and no arguments are given, the command SHALL print help and SHALL exit 0 without copying. `c3po clippy help` SHALL print that same help and SHALL exit 0. `c3po clippy version` SHALL print the c3po version and SHALL exit 0. There SHALL be no clippy screen. The usage record SHALL NOT store the text.

#### Scenario: Stdin is copied and not printed
- **WHEN** stdin is two lines and the operator runs `c3po clippy`
- **THEN** the process SHALL exit 0
- **AND** stdout SHALL be exactly `copied`
- **AND** the clipboard SHALL receive those exact bytes, including the line break
- **AND** stdout and stderr SHALL NOT contain the text

#### Scenario: Arguments are one string
- **WHEN** the operator runs `c3po clippy hello world`
- **THEN** the clipboard SHALL receive `hello world`

#### Scenario: A newline inside an argument is kept
- **WHEN** one argument contains a newline
- **THEN** the clipboard SHALL receive that newline

#### Scenario: An empty text is refused
- **WHEN** stdin is empty and no arguments are given
- **THEN** the process SHALL exit 2
- **AND** nothing SHALL be copied
- **AND** stderr SHALL name the usage

#### Scenario: A missing wl-copy is an error
- **WHEN** `wl-copy` is not on `PATH`
- **THEN** the process SHALL exit 1
- **AND** stderr SHALL be `wl-copy is required.`
- **AND** the text SHALL NOT be printed

#### Scenario: A terminal with no text prints help
- **WHEN** stdin is a terminal and no arguments are given
- **THEN** the process SHALL exit 0
- **AND** the text SHALL name `c3po clippy`
- **AND** nothing SHALL be copied

#### Scenario: Help and version
- **WHEN** the operator runs `c3po clippy help`
- **THEN** the process SHALL exit 0
- **AND** the text SHALL name `c3po clippy`
- **WHEN** the operator runs `c3po clippy version`
- **THEN** stdout SHALL start with `c3po `

