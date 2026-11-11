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
The header SHALL end with ` sha256 <hex>`, the 64 lowercase hex digit SHA-256
of the whole file. The tool SHALL scan the file once and keep only the lines it
prints. A `--strict-lines` range longer than `max_read_lines` MAY fail before
the file is read; a `--strict-lines` range past the last line SHALL fail after
the scan.
With `--number`, each content line SHALL be prefixed with its absolute line
number.

#### Scenario: Bounded slice
- **WHEN** the user runs `rgw-ast read f.go --lines 1-10`
- **THEN** stdout SHALL contain at most 10 lines of file content
- **AND** the header SHALL contain `of <total>`
- **AND** the header SHALL contain `sha256 <hex>` of the whole file, equal to
  `rgw-ast hash f.go`

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

### Requirement: Cyclomatic gate on plain Go patch
A `rgw-ast patch` without `--symbol` SHALL, after the hash matches and the
replacements are applied in memory and before the file is written, score
every Go function declaration in the old and the new bytes when the file
name ends in `.go`. `apply`, `create`, `append`, and `patch --symbol` SHALL
NOT be gated.

A declaration SHALL be named as cyclotree names it: the receiver type with a
leading pointer star removed, `.`, and the function name; a function with no
receiver is its name. Function literals SHALL NOT be scored on their own and
SHALL NOT add their body to the enclosing declaration. The score SHALL be the
cyclomatic number cyclotree reports for that declaration, from one shared
counter.

The old and the new bytes are a clean parse when go/parser accepts them and
no two declarations share a name. When either is not a clean parse, the file
SHALL be written and stdout SHALL be the ok line followed by
`cc  <relpath>  unparsed`, where `<relpath>` is the slash path the ok line
prints.

When both are clean, a declaration is changed when its score changed, it
exists only in the new bytes, or it exists only in the old bytes. The patch
SHALL be refused when any declaration moves from below `max_cc` to `max_cc`
or above; a declaration only in the new bytes counts as below before the
edit. A declaration already at or above `max_cc` MAY rise. A removal SHALL
NOT refuse.

When not refused, the file SHALL be written and stdout SHALL be the ok line
`ok  <hash>  <relpath>` followed by one line per changed declaration sorted
by name: `cc  <relpath>:<Name>  <old> -> <new>`,
`cc  <relpath>:<Name>  new -> <score>`, or `cc  <relpath>:<Name>  <score> -> gone`.

When refused, nothing SHALL be written, no ok line and no cc line SHALL be
printed, stdout SHALL hold one line per refusing declaration sorted by name,
`refused  <relpath>:<Name>  cc <old> -> <new> crosses <max_cc>` or
`refused  <relpath>:<Name>  cc new -> <score> crosses <max_cc>`, stderr SHALL
NOT hold an `Error:` line, and the process SHALL exit 1.

A hash mismatch, a missing match, or a usage error SHALL behave as before and
SHALL print no cc line.

#### Scenario: Rise under max_cc is written with a cc line
- **GIVEN** `f.go` holds function `F` with cyclomatic number 5 and `max_cc` is 30
- **WHEN** a plain patch raises `F` to 6
- **THEN** the file SHALL be written
- **AND** stdout SHALL be `ok  <hash>  f.go` then `cc  f.go:F  5 -> 6`

#### Scenario: Crossing max_cc is refused
- **GIVEN** function `F` in `f.go` has cyclomatic number 29 and `max_cc` is 30
- **WHEN** a plain patch raises `F` to 30
- **THEN** the file SHALL NOT be written
- **AND** stdout SHALL be exactly `refused  f.go:F  cc 29 -> 30 crosses 30`
- **AND** stderr SHALL NOT contain `Error:`
- **AND** the process SHALL exit 1

#### Scenario: Already-high function may rise
- **GIVEN** function `F` in `f.go` has cyclomatic number 30 and `max_cc` is 30
- **WHEN** a plain patch raises `F` to 33
- **THEN** the file SHALL be written
- **AND** stdout SHALL end with `cc  f.go:F  30 -> 33`

#### Scenario: New function at or above max_cc is refused
- **GIVEN** `max_cc` is 30
- **WHEN** a plain patch adds function `B` with cyclomatic number 31 to `f.go`
- **THEN** the file SHALL NOT be written
- **AND** stdout SHALL be exactly `refused  f.go:B  cc new -> 31 crosses 30`
- **AND** the process SHALL exit 1

#### Scenario: Removed function is written
- **WHEN** a plain patch removes function `G` with cyclomatic number 3 from `f.go`
- **THEN** the file SHALL be written
- **AND** stdout SHALL end with `cc  f.go:G  3 -> gone`

#### Scenario: Non-Go file has no cc line
- **WHEN** a plain patch of `f.py` or `f.txt` adds many branches
- **THEN** the file SHALL be written
- **AND** stdout SHALL be only the ok line

#### Scenario: Unparsed Go file is written
- **GIVEN** `f.go` is rejected by go/parser
- **WHEN** a plain patch matches and applies
- **THEN** the file SHALL be written
- **AND** stdout SHALL be the ok line then `cc  f.go  unparsed`

#### Scenario: Hash mismatch prints no cc line
- **WHEN** a plain patch of a Go file gives a stale `--expect-hash`
- **THEN** nothing SHALL be written
- **AND** stdout SHALL hold no cc or refused line
- **AND** the process SHALL exit 1 with the hash mismatch error

#### Scenario: Symbol patch is not gated
- **GIVEN** function `F` has cyclomatic number 29 and `max_cc` is 30
- **WHEN** `patch --symbol F` raises `F` to 30
- **THEN** the file SHALL be written with its formatter line and no cc line

