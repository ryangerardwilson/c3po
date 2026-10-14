## MODIFIED Requirements

### Requirement: Bounded read
`rgw-ast read <file> [--lines <start>-<end>]` SHALL print text from the file.
When `--lines` is omitted under enforcement, whole-file read of non-binary text
SHALL fail. With `--lines`, the range is 1-based inclusive.

When the requested range length exceeds `max_read_lines`, the tool SHALL by
default clamp to `max_read_lines` lines from the start of the range, print those
lines, and emit a continuation hint including `next_start` rather than failing
hard. Flag `--strict-lines` MAY restore hard fail.

With `--number`, each content line SHALL be prefixed with its absolute line
number. Output SHOULD begin with a header line identifying path and range.

#### Scenario: Bounded slice
- **WHEN** the user runs `rgw-ast read f.go --lines 1-10`
- **THEN** stdout SHALL contain at most 10 lines of file content

#### Scenario: Whole-file read denied when enforced
- **GIVEN** enforcement is active and `deny_whole_file_read` is true
- **WHEN** the user runs `rgw-ast read src/main.go` without `--lines`
- **THEN** the process SHALL exit non-zero
- **AND** stderr SHALL direct the user to `--lines` or intelligence commands

#### Scenario: Whole-file read denied for json when enforced
- **GIVEN** enforcement is active and `deny_whole_file_read` is true
- **WHEN** the user runs `rgw-ast read package.json` without `--lines`
- **THEN** the process SHALL exit non-zero
- **AND** stderr SHALL direct the user to `--lines` or intelligence commands

#### Scenario: Clamp oversized range
- **GIVEN** max_read_lines is 200
- **WHEN** the user runs `rgw-ast read f.go --lines 1-260`
- **THEN** the process SHALL exit 0
- **AND** at most 200 content lines SHALL be printed
- **AND** a next_start continuation hint SHALL be available

### Requirement: Hash-guarded exact patch
`rgw-ast patch <file> --expect-hash <sha256>` SHALL apply one or more exact
replacements when the file hash matches. Content sources:
- `--old` / `--new` argv pairs (single op)
- `--old-file` / `--new-file`
- `--ops-file <json>` as an array of `{"old":"...","new":"..."}` applied in
  order against the original content, then one atomic write

If the hash mismatches, old string is not found, or old string matches more
than once for an op, the tool SHALL fail without modifying the file.
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

#### Scenario: Ops file batch
- **GIVEN** two unique replacements and hash H
- **WHEN** patch runs with `--ops-file` listing both against H
- **THEN** both replacements SHALL apply in one write
