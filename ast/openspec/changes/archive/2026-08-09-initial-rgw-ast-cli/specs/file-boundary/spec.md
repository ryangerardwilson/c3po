## ADDED Requirements

### Requirement: Hash command
`rgw-ast hash <file> [<file>...]` SHALL print SHA-256 hex digests for each
path (workspace-relative or absolute under root) without printing file
contents. Output format SHALL be `hash  path` per line (hash first).

#### Scenario: Hash one file
- **GIVEN** a readable file under the workspace root
- **WHEN** the user runs `rgw-ast hash <path>`
- **THEN** stdout SHALL contain one line with the SHA-256 hex and path
- **AND** the process SHALL exit 0

### Requirement: Bounded read
`rgw-ast read <file> --lines <start>-<end>` SHALL print only the inclusive
line range requested. Line numbers are 1-based. The range length SHALL NOT
exceed `max_read_lines` from global config.

When enforcement is active and `deny_whole_file_read` is true, `rgw-ast read`
without `--lines` SHALL fail for included source files.

#### Scenario: Bounded slice
- **WHEN** the user runs `rgw-ast read f.go --lines 1-10`
- **THEN** stdout SHALL contain at most 10 lines of file content

#### Scenario: Whole-file read denied when enforced
- **GIVEN** enforcement is active and `deny_whole_file_read` is true
- **WHEN** the user runs `rgw-ast read src/main.go` without `--lines`
- **THEN** the process SHALL exit non-zero
- **AND** stderr SHALL direct the user to `--lines` or intelligence commands

### Requirement: Hash-guarded exact patch
`rgw-ast patch <file> --expect-hash <sha256> --old <old> --new <new>` SHALL
replace exactly one occurrence of the old string with the new string when the
file's current SHA-256 matches `--expect-hash`. On success it SHALL write
atomically and print the new hash. If the hash mismatches, old string is not
found, or old string matches more than once, the tool SHALL fail without
modifying the file.

When enforcement is active and `require_hash_before_patch` is true, omitting
`--expect-hash` SHALL be a usage or runtime error.

#### Scenario: Successful patch
- **GIVEN** a file with unique contents matching `--old` and hash H
- **WHEN** patch runs with `--expect-hash H`
- **THEN** the file content SHALL reflect the replacement
- **AND** stdout SHALL report the new SHA-256

#### Scenario: Stale hash rejects write
- **GIVEN** the file hash is not H
- **WHEN** patch runs with `--expect-hash H`
- **THEN** the file MUST remain unchanged
- **AND** the process SHALL exit 1
