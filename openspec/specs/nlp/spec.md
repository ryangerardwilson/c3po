# nlp Specification

## Purpose

The neuro-linguistic programming trainer is the c3po tui nlp screen, and its courses stay in the sqlx database nlp.
## Requirements
### Requirement: The trainer is a c3po screen
`c3po tui nlp` SHALL be the only command that opens the neuro-linguistic programming trainer. `c3po nlp` SHALL exit 2. Courses SHALL stay in the sqlx database `nlp` and SHALL NOT move into the sqlx database `c3po`. The screen SHALL show one flat list of practice sessions. Each session SHALL be one page. Opening a session SHALL enter the touch-typing interface for that page and SHALL NOT show a part menu, a section menu, or a document menu. On the list, `n` SHALL create a session, `,rn` SHALL rename the selected session, and `dd` SHALL delete it. Help SHALL open and close only with `?`. The session list and the type view SHALL NOT show a shortcut string. The screen SHALL NOT offer bookmarks. An extra argument SHALL print the screen help and SHALL exit 2. `c3po tui nlp help` SHALL exit 0 and SHALL say neuro-linguistic programming. `c3po tui nlp version` SHALL print the c3po version. A pipe with no `--resume` id SHALL NOT write a session. On a terminal, the screen SHALL store its place in the sqlx database `c3po`, including the course identity and the menu position, and SHALL NOT store the course markdown.

#### Scenario: Help names the trainer
- **WHEN** the operator runs `c3po tui nlp help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL say neuro-linguistic programming
- **AND** `c3po nlp` SHALL exit 2

#### Scenario: Extra arguments are rejected
- **WHEN** the operator runs `c3po tui nlp bookmarks`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL include the screen help
- **AND** the text SHALL NOT offer bookmarks

#### Scenario: A pipe does not store a place
- **GIVEN** stdout is not a terminal
- **WHEN** `c3po tui nlp` runs with no `--resume` id
- **THEN** it MUST NOT write a TUI session

#### Scenario: Opening a session starts typing
- **GIVEN** the practice list shows a session
- **WHEN** the operator opens that session
- **THEN** the touch-typing interface for that page MUST be showing
- **AND** a part menu, a section menu, or a document menu MUST NOT be shown

#### Scenario: The place names the course
- **GIVEN** the trainer is open on a terminal with a course selected
- **WHEN** it stores its place
- **THEN** the record MUST name the screen `nlp` and identify that course
- **AND** the record MUST NOT contain the course markdown
