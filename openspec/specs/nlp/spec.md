# nlp Specification

## Purpose

The neuro-linguistic programming trainer is the c3po tui nlp screen, and its courses stay in the sqlx database nlp.
## Requirements
### Requirement: The trainer is a c3po screen
`c3po tui nlp` SHALL be the only command that opens the neuro-linguistic programming trainer. `c3po nlp` SHALL exit 2. Courses SHALL stay in the sqlx database `nlp` and SHALL NOT move into the sqlx database `c3po`. The screen SHALL show one flat list of practice sessions. Each session SHALL be one page. Opening a session SHALL enter the touch-typing interface for that page and SHALL NOT show a part menu, a section menu, or a document menu. On the list, `n` SHALL ask for a name and then open vim for that session's text. Saving text in vim SHALL add the session, and that page SHALL be the saved text. Quitting vim without text SHALL NOT add a session. `e` SHALL open the selected session's text in vim and SHALL store the saved text on that page. `,rn` SHALL rename the selected session, and `dd` SHALL delete it. On the session list, help SHALL open and close only with `?`. During touch typing, `?` SHALL be typed and SHALL NOT open help. The session list and the type view SHALL NOT show a shortcut string. The screen SHALL NOT offer bookmarks. An extra argument SHALL print the screen help and SHALL exit 2. `c3po tui nlp help` SHALL exit 0 and SHALL say neuro-linguistic programming. `c3po tui nlp version` SHALL print the c3po version. A pipe with no `--resume` id SHALL NOT write a session. On a terminal, the screen SHALL store its place in the sqlx database `c3po`, including the course identity and the menu position, and SHALL NOT store the course markdown.

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

#### Scenario: A question mark types during touch typing
- **GIVEN** the touch-typing interface is showing
- **WHEN** the operator presses `?`
- **THEN** help MUST NOT open
- **AND** the key MUST be typed

#### Scenario: n opens vim for the page text
- **GIVEN** the operator is on the session list
- **WHEN** the operator presses `n`, enters a name, and saves text in vim
- **THEN** nlp MUST add that session
- **AND** that session MUST be one page
- **AND** that page MUST be the saved text

#### Scenario: an empty vim buffer does not create a session
- **GIVEN** the operator is on the session list
- **WHEN** the operator presses `n`, enters a name, and quits vim without text
- **THEN** nlp MUST NOT add that session

#### Scenario: e edits the selected session in vim
- **GIVEN** the practice list shows a session
- **WHEN** the operator presses `e` and saves new text in vim
- **THEN** that session's page MUST be the saved text
- **AND** other sessions MUST keep their text

#### Scenario: The place names the course
- **GIVEN** the trainer is open on a terminal with a course selected
- **WHEN** it stores its place
- **THEN** the record MUST name the screen `nlp` and identify that course
- **AND** the record MUST NOT contain the course markdown
