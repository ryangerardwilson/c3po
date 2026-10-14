## ADDED Requirements

### Requirement: Guarded create
`rgw-ast create <file> --expect-absent` SHALL create a new file only when the
path does not exist. Content SHALL come from `--from-file <path>` or `--stdin`.
If the path exists, the command SHALL fail without modifying it. With
`--parents`, missing parent directories SHALL be created. On success stdout
SHALL report the new SHA-256 and path.

#### Scenario: Create absent file
- **GIVEN** path `new.md` does not exist under the workspace root
- **WHEN** `rgw-ast create new.md --expect-absent --from-file content.md`
- **THEN** the file SHALL exist with that content
- **AND** stdout SHALL include the new hash

#### Scenario: Create refuses existing
- **GIVEN** path `exists.go` already exists
- **WHEN** create runs with `--expect-absent`
- **THEN** the process SHALL exit non-zero
- **AND** the file content MUST remain unchanged

### Requirement: Guarded append
`rgw-ast append <file> --expect-hash <sha>` SHALL append bytes from `--stdin`
or `--from-file` only when the current hash matches, and SHALL write atomically.

#### Scenario: Append with matching hash
- **GIVEN** a file with hash H
- **WHEN** append runs with `--expect-hash H` and stdin content
- **THEN** the file SHALL end with that content
- **AND** stdout SHALL report the new hash
