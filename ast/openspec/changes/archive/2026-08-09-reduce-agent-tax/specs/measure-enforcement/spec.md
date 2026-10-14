## MODIFIED Requirements

### Requirement: LOC measure
`rgw-ast measure` and the measure embedded in `status` and enforcement checks
SHALL count physical lines of text files under the workspace root that match
global include globs and do not match exclude globs. Files containing a NUL
byte SHALL be skipped.

Results SHALL be cached under the global cache directory using a fingerprint of
the root path, include/exclude policy, and a root content fingerprint (root
directory mtime and git HEAD when present). A cache entry younger than 60
seconds with a matching fingerprint SHALL be reused without walking the tree.
Commands that need enforcement SHALL use the cached path rather than forcing a
full recount on every invocation.

#### Scenario: Exclude node_modules
- **GIVEN** source files exist under `node_modules/`
- **WHEN** measure runs with default excludes
- **THEN** those files MUST NOT contribute to the line count

#### Scenario: Cache hit avoids full walk
- **GIVEN** a successful measure for a root was cached with matching fingerprint
  within 60 seconds
- **WHEN** `rgw-ast status` runs again for the same root and policy
- **THEN** the tool SHALL return the cached loc without requiring a full tree walk

## ADDED Requirements

### Requirement: Explicit workspace root flag
`rgw-ast` SHALL accept a global `--root <path>` flag that sets the workspace
root to the absolute form of that path (must exist and be a directory) instead
of discovering git root from cwd.

#### Scenario: --root scopes measure
- **GIVEN** a subdirectory of a large monorepo
- **WHEN** the user runs `rgw-ast status --root <subdir>`
- **THEN** reported root SHALL be that subdirectory
- **AND** loc SHALL be measured only under that path
