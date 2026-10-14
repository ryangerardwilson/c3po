## ADDED Requirements

### Requirement: Workspace root discovery
`rgw-ast` SHALL resolve the workspace root as the nearest ancestor directory
containing a `.git` entry (file or directory), walking from the process working
directory. If none exists, the absolute working directory SHALL be the root.

#### Scenario: Git root
- **GIVEN** the cwd is inside a git repository
- **WHEN** `rgw-ast status` runs
- **THEN** the reported root SHALL be that repository's git root

#### Scenario: Non-git directory
- **GIVEN** the cwd is not inside a git repository
- **WHEN** `rgw-ast status` runs
- **THEN** the reported root SHALL be the absolute cwd

### Requirement: LOC measure
`rgw-ast measure` and the measure embedded in `status` SHALL count physical
lines of text files under the workspace root that match global include globs
and do not match exclude globs. Files containing a NUL byte SHALL be skipped.

#### Scenario: Exclude node_modules
- **GIVEN** source files exist under `node_modules/`
- **WHEN** measure runs with default excludes
- **THEN** those files MUST NOT contribute to the line count

### Requirement: Enforcement decision
Enforcement SHALL be derived only from global config and measured LOC:

- `enforcement.mode = "always"` → enforced is true
- `enforcement.mode = "never"` → enforced is false
- `enforcement.mode = "auto"` → enforced is true iff `loc >= threshold_loc`

#### Scenario: Auto above threshold
- **GIVEN** mode is `auto`, threshold is 5000, and measured loc is 5000 or more
- **WHEN** `rgw-ast status` runs
- **THEN** output SHALL report `enforced: true` (or JSON `enforced: true`)

#### Scenario: Auto below threshold
- **GIVEN** mode is `auto`, threshold is 5000, and measured loc is 4999 or less
- **WHEN** `rgw-ast status` runs
- **THEN** output SHALL report `enforced: false`

### Requirement: Status and measure output
`rgw-ast status` SHALL report at least: root, loc, threshold_loc, mode, and
enforced. `rgw-ast measure` SHALL report at least root and loc. Both SHALL
accept `--json` for a single JSON object on stdout.

#### Scenario: Status JSON
- **WHEN** the user runs `rgw-ast status --json`
- **THEN** stdout SHALL be one JSON object including keys `root`, `loc`,
  `threshold_loc`, `mode`, and `enforced`
