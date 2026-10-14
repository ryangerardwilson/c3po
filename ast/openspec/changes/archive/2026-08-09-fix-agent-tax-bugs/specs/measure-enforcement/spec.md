## MODIFIED Requirements

### Requirement: LOC measure
`rgw-ast measure` and the measure embedded in `status` and enforcement checks
SHALL count physical lines of text files under the workspace root that match
global include globs and do not match exclude globs. Files containing a NUL
byte SHALL be skipped.

When the workspace root is a Git repository, measure SHALL by default:
- skip nested directories that contain their own `.git` entry (nested repos)
- skip paths matching the root repository's `.gitignore` and `.git/info/exclude`
  patterns when those files can be loaded

Results SHALL be cached under the global cache directory using a fingerprint of
the root path, include/exclude policy, scope rules, and a root content
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

#### Scenario: Cache hit avoids full walk
- **GIVEN** a successful measure for a root was cached with matching fingerprint
  within 60 seconds
- **WHEN** `rgw-ast status` runs again for the same root and policy without
  `--refresh`
- **THEN** the tool SHALL return the cached loc without requiring a full tree walk

#### Scenario: Refresh bypasses cache
- **WHEN** the user runs `rgw-ast status --refresh`
- **THEN** the tool SHALL recompute loc even if a warm cache entry exists

## ADDED Requirements

### Requirement: Status cache and scope diagnostics
`rgw-ast status --json` SHALL include `cache_hit` (boolean) and when available
`cache_age_ms` (number). It SHOULD include `nested_repos_skipped` and
`gitignore_active` when measured.

#### Scenario: Status JSON cache fields
- **WHEN** status runs with `--json` after a warm cache hit
- **THEN** the object SHALL contain `"cache_hit": true`
