## MODIFIED Requirements

### Requirement: Flat verb CLI surface
The `rgw-ast` binary SHALL expose a flat verb grammar for agent and human use.

Supported top-level commands SHALL include: `help`, `version`, `config`,
`status`, `measure`, `map`, `show`, `search`, `hash`, `read`, `patch`,
`create`, `append`, `delete`, `move`, `apply`, `explain`, `hook`, `exec`,
`doctor`, `agents-block`, `packages`, `affected`, `graph`, `neighbors`, `cut`.
`--version` and `-v` SHALL behave as `version`.

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

#### Scenario: Version flag
- **WHEN** the user runs `rgw-ast --version`
- **THEN** stdout SHALL contain the version string
- **AND** the process SHALL exit 0

## ADDED Requirements

### Requirement: Absolute paths choose their root
When `--root` is not given and a file verb (`hash`, `read`, `create`,
`append`, `patch`, `delete`, `move`, `map`, `explain`) receives an absolute
path outside the root resolved from the current directory, the verb SHALL
use the workspace root resolved from that path (its nearest git worktree, or
its directory). With `--root`, a path outside that root SHALL still fail.

#### Scenario: Read a file in another repository
- **GIVEN** the current directory is repository A
- **WHEN** the user runs `rgw-ast read /abs/path/in/B/f.go --lines 1-5`
- **THEN** the read SHALL succeed using repository B's enforcement
