## ADDED Requirements

### Requirement: A screen can resume
Each screen `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, and `looptab` SHALL store its place in the sqlx database `c3po` when it is opened on a terminal or with `--resume <id>`. The stored place SHALL include the screen, the workspace directory, and the cursor or path the operator had open. It MUST NOT include a pass secret. `--resume <id>` SHALL restore that place. A pipe with no `--resume` id SHALL NOT write a session. Closing the screen SHALL drop the active record and SHALL keep the stored place. A missing id, a bad id, or `--resume` given twice SHALL fail without opening the screen.

#### Scenario: Resume returns to the stored place

- **GIVEN** screen `o` stored a directory and a selected path
- **WHEN** the operator runs `c3po tui o --resume <id>`
- **THEN** the navigator SHALL open in that directory at that path

#### Scenario: A pipe does not write a session

- **GIVEN** stdout is not a terminal
- **WHEN** `c3po tui l` runs with no `--resume` id
- **THEN** it MUST NOT write a TUI session

#### Scenario: Pass keeps the secret out of the session

- **GIVEN** the pass screen is showing an item
- **WHEN** it stores its place
- **THEN** the record MUST identify the vault and the item
- **AND** it MUST NOT contain the item secret
