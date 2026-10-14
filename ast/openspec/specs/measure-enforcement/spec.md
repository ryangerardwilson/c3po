# measure-enforcement Specification

## Purpose
Resolve the workspace root, measure its code lines within the shared walk scope, and decide from global policy whether the boundary is enforced.
## Requirements
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
`rgw-ast measure` and the measure embedded in `status` and enforcement checks
SHALL count physical lines of text files under the workspace root that match
global include globs and do not match exclude globs. Files containing a NUL
byte SHALL be skipped.

Lines of included files that also match a global `prose` glob SHALL be
reported separately as `prose_loc`. They SHALL NOT count toward `loc` unless
`enforcement.count_prose` is true. `file_count` SHALL include prose files.

When the workspace root is a Git repository, measure SHALL by default:
- skip nested directories that contain their own `.git` entry, directory or
  file (nested repos and linked worktrees)
- skip paths matching the root repository's `.gitignore` and `.git/info/exclude`
  patterns when those files can be loaded
- skip a directory whose own `.gitignore` is the single rule `*`

Results SHALL be cached under the global cache directory using a fingerprint of
the root path, include/exclude/prose policy, scope rules, and a root content
fingerprint. A cache entry younger than 60 seconds with a matching fingerprint
SHALL be reused unless `--refresh` is set. Commands that need enforcement SHALL
use the cached path rather than forcing a full recount on every invocation.

#### Scenario: Exclude node_modules
- **GIVEN** source files exist under `node_modules/`
- **WHEN** measure runs with default excludes
- **THEN** those files MUST NOT contribute to the line count

#### Scenario: Nested git root skipped
- **GIVEN** workspace root is a git repo containing `Apps/foo/.git`
- **WHEN** measure runs without filesystem scope override
- **THEN** files under `Apps/foo/` MUST NOT contribute to loc

#### Scenario: Prose does not reach the threshold
- **GIVEN** a repository with 6000 lines of Markdown and 100 lines of Go
- **AND** `enforcement.count_prose` is false
- **WHEN** `rgw-ast status --json` runs
- **THEN** `loc` SHALL be 100 and `prose_loc` SHALL be 6000
- **AND** `enforced` SHALL be false in `auto` mode at threshold 5000

#### Scenario: Cache hit avoids full walk
- **GIVEN** a successful measure for a root was cached with matching fingerprint
  within 60 seconds
- **WHEN** `rgw-ast status` runs again for the same root and policy without
  `--refresh`
- **THEN** the tool SHALL return the cached loc without requiring a full tree walk

#### Scenario: Refresh bypasses cache
- **WHEN** the user runs `rgw-ast status --refresh`
- **THEN** the tool SHALL recompute loc even if a warm cache entry exists

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

### Requirement: Explicit workspace root flag
`rgw-ast` SHALL accept a global `--root <path>` flag that sets the workspace
root to the absolute form of that path (must exist and be a directory) instead
of discovering git root from cwd.

#### Scenario: --root scopes measure
- **GIVEN** a subdirectory of a large monorepo
- **WHEN** the user runs `rgw-ast status --root <subdir>`
- **THEN** reported root SHALL be that subdirectory
- **AND** loc SHALL be measured only under that path

### Requirement: Status cache and scope diagnostics
`rgw-ast status --json` SHALL include `cache_hit` (boolean) and when available
`cache_age_ms` (number). It SHOULD include `nested_repos_skipped` and
`gitignore_active` when measured.

#### Scenario: Status JSON cache fields
- **WHEN** status runs with `--json` after a warm cache hit
- **THEN** the object SHALL contain `"cache_hit": true`

