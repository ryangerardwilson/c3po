## MODIFIED Requirements

### Requirement: Flat verb CLI surface
The `rgw-ast` binary SHALL expose a flat verb grammar for agent and human use.

Supported top-level commands SHALL include: `help`, `version`, `config`,
`status`, `measure`, `map`, `show`, `search`, `hash`, `read`, `patch`,
`create`, `append`, `delete`, `explain`, `hook`, `exec`, `doctor`,
`agents-block`, `packages`, `affected`, `graph`, `neighbors`, `cut`.

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

### Requirement: Affected verb
`rgw-ast affected` SHALL map changed paths onto discovered packages.
With no path arguments it SHALL use Git paths dirty versus HEAD plus
untracked non-ignored files. With path arguments it SHALL use those paths.
`--json` SHALL include `paths`, `packages`, and `unmapped`. Exit 0 when
nothing is affected.

Mapping SHALL be the union of filesystem prefix matching and reverse
workspace Go-import reachability: if package B imports package A and A is
dirty, B's discovered app/test package SHALL be included.

#### Scenario: Map a dirty app file
- **GIVEN** path `Apps/tm/foo.go`
- **WHEN** `rgw-ast affected --json Apps/tm/foo.go` runs
- **THEN** packages MUST include `Apps/tm`

#### Scenario: Reverse import reachability
- **GIVEN** workspace module B imports workspace module A
- **WHEN** `rgw-ast affected --json` is given a Go file in A
- **THEN** packages MUST include B's discovered app package

#### Scenario: Nothing affected
- **GIVEN** a clean Git worktree and no path arguments
- **WHEN** `rgw-ast affected --json` runs
- **THEN** packages MUST be empty
- **AND** the process SHALL exit 0

## ADDED Requirements

### Requirement: Graph verbs
`rgw-ast graph` SHALL print a summary of the workspace connection graph
(optional `--json`): node counts by kind, edge counts by relation, connected
component count, and highest-degree Go package dirs.

`rgw-ast neighbors <id>` SHALL print the 1-hop neighborhood of a node
(optional `--json`). IDs are `app|test|desktop|gopkg|file` plus a path, or a
path that uniquely resolves. JSON SHALL include neighbor ids and a
`ring_hash` over the sorted 1-ring.

`rgw-ast cut [path]` SHALL list Go-package-dir articulation points of the
undirected import graph, or report whether `path`'s package dir is one
(optional `--json`).

#### Scenario: Neighbors of a Go file
- **GIVEN** a Go file that imports another workspace package
- **WHEN** `rgw-ast neighbors --json` runs on that file
- **THEN** the neighborhood MUST include an `imports` edge to that package dir
- **AND** `ring_hash` MUST be a non-empty hex digest

#### Scenario: Graph summary
- **GIVEN** a workspace with at least one Go module
- **WHEN** `rgw-ast graph --json` runs
- **THEN** stdout MUST include node and edge counts
- **AND** component count MUST be at least 1
