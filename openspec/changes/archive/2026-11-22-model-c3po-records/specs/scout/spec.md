## MODIFIED Requirements

### Requirement: A stored-data migration runs once
A change to the c3po database schema or to stored c3po data SHALL ship as a numbered migration in one ordered list. Opening the database SHALL apply, in order, every migration whose version is newer than the newest version in `schema_migration`. A migration's sql and its `schema_migration` row SHALL commit in one transaction. When the sql fails, neither SHALL commit, the command SHALL report the error, and a later open SHALL try that migration again. A migration SHALL be safe to apply twice. An applied migration SHALL NOT be edited. The database SHALL run with foreign keys enforced.

#### Scenario: A migration runs one time
- **WHEN** the database is opened twice
- **THEN** each migration SHALL apply once
- **AND** `schema_migration` SHALL hold one row for each version

#### Scenario: A failed migration commits nothing
- **WHEN** a migration's sql fails partway
- **THEN** no row SHALL change and its version SHALL NOT be recorded
- **AND** a later open SHALL apply that migration

#### Scenario: A record keeps its id
- **WHEN** a record with the same app, kind, and key is written twice
- **THEN** one row SHALL remain and its id SHALL NOT change

### Requirement: Drive sync state stores no parent path
Drive sync state SHALL be stored in the table `gdrive_sync_entry`, one row per entry, unique on preset, registration, and relpath. Kind SHALL be `file` or `dir`. An entry SHALL store its `relpath` and SHALL NOT store `parent_relpath`. Loading state SHALL derive each parent from the directory part of `relpath`, and an entry at the top of the folder SHALL have an empty parent. Saving state SHALL make the stored entries of that registration match the saved set in one transaction, and an entry that stays SHALL keep its row. Removing a registration SHALL delete its entries before it removes the registration from the config, and SHALL stop with the error when that delete fails. Registering SHALL clear any entries an earlier registration with the same id left. A path that is not a production record path SHALL stay a JSON file.

#### Scenario: Saved state has no parent path
- **WHEN** state holding `a/b.txt` is saved and loaded
- **THEN** no stored column SHALL hold a parent path
- **AND** the loaded parent of `a/b.txt` SHALL be `a`

#### Scenario: Old state loses its parent path
- **GIVEN** a stored state record whose entries hold `parent_relpath`
- **WHEN** the database is opened and migrations 4 and 9 run
- **THEN** each entry SHALL be a `gdrive_sync_entry` row with no parent path
- **AND** each loaded parent SHALL be derived from its relpath

#### Scenario: An entry that stays keeps its row
- **GIVEN** stored state with entries `a`, `a/b`, and `c.txt`
- **WHEN** state without `c.txt` and with a new size for `a/b` is saved
- **THEN** `c.txt` SHALL be gone
- **AND** `a` and `a/b` SHALL keep their row ids
