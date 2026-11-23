## REMOVED Requirements

### Requirement: A screen backup survives power loss
**Reason**: c3po published its own snapshot every 10 minutes from `c3po-snapshot.timer`, ran `s3 make bucket` before every publish, and kept its own restore and empty-file guard. nebula keep is now the one backup component for every application database.
**Migration**: `A screen backup trails its writes`. `c3po __snapshot` stays for one release and marks the database changed. The installer retires `c3po-snapshot.timer`; nebula installs `nebula-keep.timer`.

## ADDED Requirements

### Requirement: A screen backup trails its writes
Every write to the runtime c3po sqlite file SHALL commit and return, and SHALL then run `nebula keep mark c3po <path>`. A write SHALL NOT upload, and a failed mark SHALL NOT fail the write. nebula keep SHALL upload a consistent copy afterwards and SHALL refuse to replace the backup with an empty database. `SQLX_ROOT` SHALL NOT mark and SHALL NOT restore.

When the runtime file is missing, opening a session SHALL run `nebula keep restore c3po <path>`. When that restore fails, the open SHALL fail, SHALL NOT create a database, and SHALL NOT mark. When the backup is absent, nebula keep creates an empty database, which c3po migrates.

`c3po __snapshot` SHALL mark the runtime database changed when it exists, and SHALL NOT create a database when the runtime file is missing. The installer SHALL NOT install `c3po-snapshot.timer`, and SHALL disable and remove it when present.

#### Scenario: A key does not wait on a backup
- **WHEN** a screen stores a new place
- **THEN** the runtime database SHALL contain that place
- **AND** the write SHALL NOT wait on an upload

#### Scenario: A failed restore does not create a database
- **GIVEN** the runtime file is missing and `nebula keep restore` fails
- **WHEN** a session open runs
- **THEN** the open SHALL fail
- **AND** the runtime file SHALL stay missing

#### Scenario: The schema is checked once per process
- **WHEN** a process writes the runtime database twice
- **THEN** the second write SHALL NOT read the migration ledger again

### Requirement: A screen never waits before it paints
A key on a screen SHALL change only the screen's own state (cursor, status, and pending intents) and SHALL paint before any write, subprocess, or network call it implies. That work SHALL run on the screen's one serial worker, in order, with a timeout. While it runs, the painted frame SHALL show the intent, and the other keys SHALL keep working. When it succeeds, the owner's answer SHALL replace the screen's copy whole; when it fails, the intent SHALL be dropped, the view SHALL return to the owner's last answer, and the status SHALL show the failure. A screen SHALL NOT edit its copy of an owner's data. The session place SHALL be written in the background and only when it changes. Leaving a screen SHALL wait at most 10 seconds for pending local writes and SHALL NOT wait on a backup. A deliberate terminal hand-off such as vim or a foreground job MAY hold the terminal; the reload after it SHALL run on the worker. A screen's own local file operation, such as o's delete and paste or the spreadsheet's save, MAY run on the key path; a command a screen runs as a process, such as o's `:` commands, SHALL NOT.

#### Scenario: A slow owner does not freeze the screen
- **GIVEN** the owner takes several seconds to answer a write
- **WHEN** the operator presses the key for that write and then moves the cursor
- **THEN** a frame showing the intent SHALL be painted before the owner answers
- **AND** the cursor SHALL move before the owner answers

#### Scenario: A failed write restores the view
- **WHEN** the owner refuses a write
- **THEN** the row SHALL return to the view
- **AND** the status SHALL show the failure
