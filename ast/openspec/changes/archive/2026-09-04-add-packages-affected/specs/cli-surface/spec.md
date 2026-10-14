## MODIFIED Requirements

### Requirement: Flat verb CLI surface
The `rgw-ast` binary SHALL expose a flat verb grammar for agent and human use.

Supported top-level commands SHALL include: `help`, `version`, `config`,
`status`, `measure`, `map`, `show`, `search`, `hash`, `read`, `patch`,
`create`, `append`, `delete`, `explain`, `hook`, `exec`, `doctor`,
`agents-block`, `packages`, `affected`.

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

## ADDED Requirements

### Requirement: Package index verb
`rgw-ast packages` SHALL discover workspace packages from the filesystem and
print them. `--json` SHALL emit a list of objects with `id`, `kind`, `path`,
`language`, optional `install`, and `verify` command strings runnable from the
workspace root.

Discovery SHALL include:

- `Apps/<name>/` when `go.mod`, `package.json`, or `install.sh` exists
- `Tests/<name>/` when `run.sh` exists
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

#### Scenario: Map a dirty app file
- **GIVEN** path `Apps/tm/foo.go`
- **WHEN** `rgw-ast affected --json Apps/tm/foo.go` runs
- **THEN** packages MUST include `Apps/tm`

#### Scenario: Nothing affected
- **GIVEN** a clean Git worktree and no path arguments
- **WHEN** `rgw-ast affected --json` runs
- **THEN** packages MUST be empty
- **AND** the process SHALL exit 0
