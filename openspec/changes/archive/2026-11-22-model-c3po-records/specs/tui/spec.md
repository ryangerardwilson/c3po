## MODIFIED Requirements

### Requirement: A screen can resume
Each screen `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, `looptab`, `nlp`, and `slack` SHALL store its place in the sqlx database `c3po` when it is opened on a terminal or with `--resume <id>`. The place SHALL be one row in `tui_session` keyed by the session id, holding the screen, the workspace directory, and the cursor or path the operator had open. It MUST NOT include a pass secret. While the screen runs, one row in `tui_active` keyed by that session id SHALL hold only the process id and its start time; it SHALL NOT repeat the screen or the workspace. That row SHALL be deleted with its session. `--resume <id>` SHALL restore that place. A pipe with no `--resume` id SHALL NOT write a session. Closing the screen SHALL drop the active row and SHALL keep the stored place. A missing id, a bad id, or `--resume` given twice SHALL fail without opening the screen.

#### Scenario: Resume returns to the stored place

- **GIVEN** screen `o` stored a directory and a selected path
- **WHEN** the operator runs `r2d2 o --resume <id>`
- **THEN** the navigator SHALL open in that directory at that path

#### Scenario: A pipe does not write a session

- **GIVEN** stdout is not a terminal
- **WHEN** `r2d2 l` runs with no `--resume` id
- **THEN** it MUST NOT write a TUI session

#### Scenario: The active row is only the process

- **WHEN** a screen is open
- **THEN** its `tui_active` row SHALL hold the process id and its start time
- **AND** the screen and workspace SHALL be read from its `tui_session` row

#### Scenario: NLP stores the course place

- **GIVEN** the nlp screen has a course selected
- **WHEN** it stores its place
- **THEN** the record MUST name the screen `nlp`
- **AND** the record MUST identify the course
- **AND** the record MUST NOT contain the course markdown

#### Scenario: Pass keeps the secret out of the session

- **GIVEN** the pass screen is showing an item
- **WHEN** it stores its place
- **THEN** the record MUST identify the vault and the item
- **AND** it MUST NOT contain the item secret

#### Scenario: Slack keeps the token out of the session

- **GIVEN** the slack screen has a workspace open
- **WHEN** it stores its place
- **THEN** the record MUST name the screen `slack`
- **AND** the record MUST identify the workspace
- **AND** it MUST NOT contain the user token

### Requirement: A screen backup survives power loss
A screen SHALL write its place into the runtime c3po sqlite file when the place changes. That write SHALL NOT publish `s3://backups/c3po.sqlite`. A snapshot SHALL publish that file about every 10 minutes, SHALL upload a consistent copy, and SHALL wait until the upload finishes or 60 seconds pass. A database in which no table other than `schema_migration` has a row SHALL NOT be published. `SQLX_ROOT` SHALL NOT publish and SHALL NOT restore.

When the runtime file is missing, opening a session SHALL load `s3://backups/c3po.sqlite` and SHALL wait up to 60 seconds. When that load fails for a reason other than the object being absent, the open SHALL fail, SHALL NOT create a database, and SHALL NOT publish. When the object is absent, an empty database MAY be created and SHALL NOT be published until it holds data.

The installer SHALL enable a user timer `c3po-snapshot.timer` with `OnActiveSec=10min`, `OnUnitActiveSec=10min`, and `AccuracySec=5s`. The timer SHALL run `c3po __snapshot`. That command SHALL publish the runtime database when it holds data, and SHALL NOT create a database when the runtime file is missing. The installer SHALL NOT run a snapshot while it installs.

#### Scenario: A key does not publish
- **WHEN** a screen stores a new place
- **THEN** the runtime database SHALL contain that place
- **AND** the backup SHALL NOT be published by that write

#### Scenario: An empty database is not published
- **GIVEN** the runtime database has no row outside `schema_migration`
- **WHEN** a snapshot runs
- **THEN** `s3://backups/c3po.sqlite` SHALL NOT be replaced

#### Scenario: A failed restore does not create a database
- **GIVEN** the runtime file is missing and the backup load fails
- **WHEN** a session open runs
- **THEN** the open SHALL fail
- **AND** the runtime file SHALL stay missing

#### Scenario: A missing runtime file loads the backup
- **GIVEN** the runtime file is missing and the backup object exists
- **WHEN** a session opens
- **THEN** the runtime file SHALL be loaded from that object before the place is read

#### Scenario: The timer publishes every 10 minutes
- **WHEN** c3po is installed
- **THEN** `c3po-snapshot.timer` SHALL set `OnActiveSec=10min` and `OnUnitActiveSec=10min`
- **AND** its service SHALL run `c3po __snapshot`

## ADDED Requirements

### Requirement: Screens outside c3po keep sessions through c3po
c3po SHALL be the one writer of screen sessions. A screen that runs outside c3po, such as the tunnel screen in nebula, SHALL keep its session through `c3po __session`, and a reader such as tesseract SHALL list running screens through it; neither SHALL read or write the session tables. `c3po __session open <screen> <pid> [<id>]` SHALL create a session, or load id, for the screen running as process pid, and SHALL print its id and, when id was given, the restored place as JSON. `note <id> <pid> <place json>` SHALL merge the place's non-empty fields. `place <id> <pid> <path> <cursor>` SHALL set the path and cursor. `close <id>` SHALL drop the active row. `active` SHALL print every running screen as `session_id`, `pid`, `pid_start`, `screen`, and `cwd`. A missing session, a session of another screen, a bad process id, or an unknown form SHALL exit 2.

#### Scenario: The tunnel screen keeps its place through c3po
- **WHEN** nebula opens a tunnel session through `c3po __session open tunnel <pid>`, places the cursor, and closes it
- **THEN** `c3po __session active` SHALL list it while it is open and SHALL NOT list it after close
- **AND** `c3po __session open tunnel <pid> <id>` SHALL print the restored place

#### Scenario: A session of another screen is refused
- **WHEN** `c3po __session open o <pid> <id>` names a tunnel session
- **THEN** the command SHALL exit 2
