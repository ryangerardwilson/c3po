## ADDED Requirements

### Requirement: Flat verb CLI surface
The `rgw-ast` binary SHALL expose a flat verb grammar for agent and human use.

Supported top-level commands SHALL include: `help`, `version`, `config`,
`status`, `measure`, `map`, `show`, `search`, `hash`, `read`, `patch`.

#### Scenario: No-args and help
- **GIVEN** the binary is installed or run from source
- **WHEN** the user runs `rgw-ast` with no arguments or `rgw-ast help`
- **THEN** the process SHALL print compact help to stdout
- **AND** the process SHALL exit 0

#### Scenario: Unknown command
- **WHEN** the user runs an unknown top-level verb
- **THEN** the process SHALL print an error to stderr
- **AND** the process SHALL exit 2

### Requirement: Deterministic exit codes
The CLI SHALL use exit code 0 for success, 1 for runtime failure, and 2 for
usage errors.

#### Scenario: Usage error
- **WHEN** required arguments for a verb are missing
- **THEN** the process SHALL exit 2

#### Scenario: Runtime error
- **WHEN** a file is missing or a patch hash mismatches
- **THEN** the process SHALL exit 1

### Requirement: Version command
`rgw-ast version` SHALL print the stamped version string and exit 0. Source
checkouts default to `0.0.0` until release stamping.

#### Scenario: Version
- **WHEN** the user runs `rgw-ast version`
- **THEN** stdout SHALL contain the version string
- **AND** the process SHALL exit 0
