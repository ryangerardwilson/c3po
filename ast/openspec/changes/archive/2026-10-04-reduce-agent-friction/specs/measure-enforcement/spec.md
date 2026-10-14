## MODIFIED Requirements

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
