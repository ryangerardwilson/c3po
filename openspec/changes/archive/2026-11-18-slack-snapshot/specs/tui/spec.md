## ADDED Requirements

### Requirement: A screen backup survives power loss
A screen SHALL write its place into the runtime c3po sqlite file when the place changes. That write SHALL NOT publish `s3://backups/c3po.sqlite`. A snapshot SHALL publish that file about every 10 minutes, SHALL upload a consistent copy, and SHALL wait until the upload finishes or 60 seconds pass. A database whose `record` table has no rows SHALL NOT be published. `SQLX_ROOT` SHALL NOT publish and SHALL NOT restore.

When the runtime file is missing, opening a session SHALL load `s3://backups/c3po.sqlite` and SHALL wait up to 60 seconds. When that load fails for a reason other than the object being absent, the open SHALL fail, SHALL NOT create a database, and SHALL NOT publish. When the object is absent, an empty database MAY be created and SHALL NOT be published until it has a record.

The installer SHALL enable a user timer `c3po-snapshot.timer` with `OnActiveSec=10min`, `OnUnitActiveSec=10min`, and `AccuracySec=5s`. The timer SHALL run `c3po __snapshot`. That command SHALL publish the runtime database when it has a record, and SHALL NOT create a database when the runtime file is missing. The installer SHALL NOT run a snapshot while it installs.

#### Scenario: A key does not publish
- **WHEN** a screen stores a new place
- **THEN** the runtime database SHALL contain that place
- **AND** the backup SHALL NOT be published by that write

#### Scenario: An empty database is not published
- **GIVEN** the runtime database has a `record` table with no rows
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
