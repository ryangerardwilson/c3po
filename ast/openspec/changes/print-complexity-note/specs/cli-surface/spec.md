## ADDED Requirements

### Requirement: Writes append a complexity note
After a successful `create`, `patch`, or `apply` that writes a Go, Python, Rust, TypeScript, JavaScript, Prisma, or SQL file, rgw-ast SHALL append the text of `cx note` for those files to stdout. A missing `cx`, a timeout, a non-zero exit, or empty output SHALL leave the command successful and SHALL leave the `ok` lines in place. `append`, `delete`, and `move` SHALL NOT append a note.

#### Scenario: A stub note is appended
- **GIVEN** `cx` on `PATH` prints a note and exits 0
- **WHEN** a successful patch writes a Go file
- **THEN** stdout SHALL contain the `ok` line
- **AND** stdout SHALL contain that note

#### Scenario: A failing cx does not fail the write
- **GIVEN** `cx` on `PATH` exits non-zero
- **WHEN** the note helper runs for a Go file
- **THEN** it SHALL print nothing
