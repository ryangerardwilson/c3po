## MODIFIED Requirements

### Requirement: Flat verb CLI surface
The `rgw-ast` binary SHALL expose a flat verb grammar for agent and human use.

Supported top-level commands SHALL include: `help`, `version`, `config`,
`status`, `measure`, `map`, `show`, `search`, `hash`, `read`, `patch`,
`create`, `append`, `delete`, `explain`, `hook`.

Global flag `--root <path>` MAY appear before the verb or immediately after it
for commands that operate on a workspace.

#### Scenario: No-args and help
- **GIVEN** the binary is installed or run from source
- **WHEN** the user runs `rgw-ast` with no arguments or `rgw-ast help`
- **THEN** the process SHALL print compact help to stdout
- **AND** the process SHALL exit 0

#### Scenario: Unknown command
- **WHEN** the user runs an unknown top-level verb
- **THEN** the process SHALL print an error to stderr
- **AND** the process SHALL exit 2
