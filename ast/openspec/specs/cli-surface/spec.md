# cli-surface Specification

## Purpose
Define the flat rgw-ast verb grammar, global flags, exit codes, version output, and the package and graph verbs that agents and humans call.
## Requirements
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

### Requirement: Package index verb
`rgw-ast packages` SHALL discover workspace packages from the filesystem and
print them. `--json` SHALL emit a list of objects with `id`, `kind`, `path`,
`language`, optional `install`, and `verify` command strings runnable from the
workspace root.

Discovery SHALL include:

- `Apps/<name>/` when `go.mod`, `package.json`, or `install.sh` exists
- `tests/<name>/` when `run.sh` exists
- `.config/hypr` and `.config/quickshell` when those directories exist

Go apps SHALL verify with `go test -C <path> ./...`. Apps with an npm `test`
script SHALL verify with `npm test --prefix <path>`. Test packages SHALL
verify by running their `run.sh`. Apps with `install.sh` SHALL set
`install` to `<path>/install.sh from <path>`.

#### Scenario: Discover a Go app
- **GIVEN** a workspace contains `Apps/tm/go.mod` and `Apps/tm/install.sh`
- **WHEN** `rgw-ast packages --json` runs
- **THEN** stdout MUST include a package with path `Apps/tm`
- **AND** verify MUST include `go test -C Apps/tm ./...`
- **AND** install MUST be `Apps/tm/install.sh from Apps/tm`

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

