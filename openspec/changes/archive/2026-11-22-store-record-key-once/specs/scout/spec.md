## ADDED Requirements

### Requirement: A stored-data migration runs once
A change to the shape of stored c3po records SHALL ship as a named migration that runs one time in the sqlx database `c3po`. The migration's sql and the `app_once` row for its name SHALL commit in one transaction. When the sql fails, neither SHALL commit, the command SHALL report the error, and a later run SHALL try the migration again. The database SHALL be published only when the migration ran. A path that is not a production record path SHALL run no migration. A migration SHALL be safe to apply twice.

#### Scenario: A migration runs one time
- **WHEN** the same migration name is run twice
- **THEN** its sql SHALL apply once

#### Scenario: A failed migration commits nothing
- **WHEN** a migration's sql fails partway
- **THEN** no record SHALL change and the name SHALL NOT be recorded
- **AND** a later run of that name SHALL apply

### Requirement: Drive sync state stores no parent path
A Drive sync state entry SHALL store its `relpath` and SHALL NOT store `parent_relpath`. Loading state SHALL derive each parent from the directory part of `relpath`, and an entry at the top of the folder SHALL have an empty parent. Before it loads state from the sqlx database, gdrive SHALL run the migration `gdrive-state-parent-1` once, which SHALL remove `parent_relpath` from every entry of every stored state.

#### Scenario: Saved state has no parent path
- **WHEN** state holding `a/b.txt` is saved and loaded
- **THEN** the stored state SHALL NOT contain `parent_relpath`
- **AND** the loaded parent of `a/b.txt` SHALL be `a`

#### Scenario: Old state loses its parent path
- **GIVEN** stored state whose entries hold `parent_relpath`
- **WHEN** gdrive loads that state
- **THEN** each loaded parent SHALL be derived from its relpath
- **AND** the stored state SHALL NOT contain `parent_relpath`
