# file-boundary Specification

## Purpose
Read bounded line ranges, hash files, and apply hash-guarded patches and transactions so no change lands on a file that changed underneath it.
## Requirements
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
`rgw-ast read <file> [--lines <start>-<end>]` SHALL print text from the file.
When `--lines` is omitted under enforcement, whole-file read of non-binary text
SHALL fail, and the error SHALL name the file's line count and the first
bounded range to request. With `--lines`, the range is 1-based inclusive.

When the requested range length exceeds `max_read_lines`, the tool SHALL by
default clamp to `max_read_lines` lines from the start of the range, print those
lines, and emit a continuation hint including `next_start` rather than failing
hard. With `--strict-lines`, a range longer than `max_read_lines` or ending
past the last line SHALL fail with exit 1.

Output SHALL begin with a header line `# <path> lines <a>-<b> of <total>`.
When the range was clamped the header SHALL add the next range to request.
With `--number`, each content line SHALL be prefixed with its absolute line
number.

#### Scenario: Bounded slice
- **WHEN** the user runs `rgw-ast read f.go --lines 1-10`
- **THEN** stdout SHALL contain at most 10 lines of file content
- **AND** the header SHALL contain `of <total>`

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

#### Scenario: Strict range past end of file
- **GIVEN** `f.go` has 195 lines
- **WHEN** the user runs `rgw-ast read f.go --lines 150-400 --strict-lines`
- **THEN** the process SHALL exit 1

### Requirement: Hash-guarded exact patch
`rgw-ast patch <file> --expect-hash <sha256>` SHALL apply one or more exact
replacements when the file hash matches. Content sources:
- `--old` / `--new` argv pairs (single op)
- `--old-file` / `--new-file`
- `--ops-file <json>` as an array of `{"old":"...","new":"...","all":bool}`
  applied in order against the original content, then one atomic write

If the hash mismatches, old string is not found, or old string matches more
than once for an op without `--all` (or `"all": true`), the tool SHALL fail
without modifying the file. A multiple-match failure SHALL list the line
number of every match. A hash mismatch SHALL say to re-hash or use the hash
printed by the last write. With `--all`, every occurrence SHALL be replaced.
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

#### Scenario: Ambiguous match lists lines
- **GIVEN** `x = 1` occurs on lines 3 and 9
- **WHEN** patch runs with `--old 'x = 1'` and no `--all`
- **THEN** the process SHALL exit 1
- **AND** stderr SHALL contain `3` and `9`

#### Scenario: Replace all
- **GIVEN** `x = 1` occurs twice
- **WHEN** patch runs with `--old 'x = 1' --new 'x = 2' --all`
- **THEN** both occurrences SHALL be replaced

### Requirement: Transactional apply
`rgw-ast apply --from-file <manifest.json|bundle.diff> [--dry-run] [--no-format]`
SHALL validate every change before writing any file and leave the tree
untouched on any failure. Ops SHALL be `edit`, `create`, `delete`, and `move`
(`move` takes `to` and `expect_hash`).

A diff source SHALL be either a bundle with `# path:` headers or a plain
unified diff (`git diff`, `diff -u`) whose paths come from its `---`/`+++`
headers (`a/` and `b/` prefixes stripped). `/dev/null` as the old side SHALL
be a create; as the new side, a delete whose removed lines must equal the
file. An edit given as a diff MAY omit `expect_hash`; its context and removed
lines SHALL then be the guard.

Go, Python, TypeScript, and declared-Markdown files SHALL be formatted unless
`--no-format` is given. Each output line SHALL report `formatter <name>` or
`formatter none`.

#### Scenario: Plain git diff applies
- **GIVEN** a file `a.txt` containing `hi`
- **WHEN** apply runs on a `git diff` that changes `hi` to `yo` with no `# path:` header
- **THEN** `a.txt` SHALL contain `yo`

#### Scenario: Stale diff context fails
- **GIVEN** a plain diff whose context no longer matches the file
- **WHEN** apply runs
- **THEN** the process SHALL exit 1 and no file SHALL change

#### Scenario: Move in a manifest
- **GIVEN** a manifest `{"files":[{"op":"move","path":"a.go","to":"b.go","expect_hash":"H"}]}`
- **AND** `a.go` has hash H and `b.go` is absent
- **WHEN** apply runs
- **THEN** `b.go` SHALL exist with the same content and `a.go` SHALL not

