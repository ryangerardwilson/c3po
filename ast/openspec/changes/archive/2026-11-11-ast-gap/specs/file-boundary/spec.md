## REMOVED Requirements

### Requirement: Cyclomatic gate on plain Go patch
**Reason**: Its scenario "Symbol patch is not gated" is no longer the contract; `patch --symbol`, `create`, and `apply` are now gated.
**Migration**: The plain patch rules and scenarios move to "Cyclomatic gate and cc lines on plain Go patch"; the other write paths are in "Cyclomatic gate on symbol patch, create, and apply".

## ADDED Requirements

### Requirement: Cyclomatic gate and cc lines on plain Go patch
A `rgw-ast patch` without `--symbol` SHALL, after the hash matches and the
replacements are applied in memory and before the file is written, score
every Go function declaration in the old and the new bytes when the file
name ends in `.go`. `patch --symbol`, `create`, and `apply` SHALL be gated by
the same refusal rule as specified in "Cyclomatic gate on symbol patch,
create, and apply". `append` SHALL NOT be gated.

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

### Requirement: Cyclomatic gate on symbol patch, create, and apply
`rgw-ast patch --symbol`, `rgw-ast create`, and `rgw-ast apply` SHALL score
the Go function declarations of every `.go` file they would write, with the
names and the counter of the plain patch gate, on the bytes that would be
written: after the formatter when that path formats (`patch --symbol`, and
`apply` unless `--no-format`), and the exact given bytes for `create`. The
old bytes of an edit SHALL be the file before the write; a created file SHALL
have no old declarations. A `move` or `delete` SHALL NOT be scored. Files not
ending in `.go` SHALL NOT be scored and SHALL NOT be refused.

The write SHALL be refused when any declaration moves from below `max_cc` to
`max_cc` or above, or a declaration only in the new bytes is at `max_cc` or
above. A declaration already at or above `max_cc` MAY rise. A removal SHALL
NOT refuse. When the old or the new bytes are not a clean parse (go/parser
rejects them or two declarations share a name), that file SHALL NOT refuse
and SHALL be written.

When refused, nothing SHALL be written: no file, no created directory, and no
other file of the same `apply` transaction. No ok line and no plan SHALL be
printed. Stdout SHALL hold one line per refusing declaration across every
file, sorted by path and then by name:
`refused  <relpath>:<Name>  cc <old> -> <new> crosses <max_cc>` or
`refused  <relpath>:<Name>  cc new -> <score> crosses <max_cc>`. Stderr SHALL
NOT hold an `Error:` line, and the process SHALL exit 1. `apply --dry-run`
SHALL refuse the same way and print the same lines instead of a planned diff.

When not refused, output SHALL be as before: the ok lines with their
formatter, and no `cc` line. A hash mismatch, an existing create path, a
missing match, or a usage error SHALL fail as before with no refused line.

#### Scenario: Symbol patch crossing max_cc is refused
- **GIVEN** function `F` in `pkg/f.go` has cyclomatic number 29 and `max_cc` is 30
- **WHEN** `patch pkg/f.go --symbol F` raises `F` to 30
- **THEN** the file SHALL NOT be written
- **AND** stdout SHALL be exactly `refused  pkg/f.go:F  cc 29 -> 30 crosses 30`
- **AND** stderr SHALL NOT contain `Error:`
- **AND** the process SHALL exit 1

#### Scenario: Symbol patch under max_cc prints no cc line
- **GIVEN** function `F` in `pkg/f.go` has cyclomatic number 5
- **WHEN** `patch --symbol F` raises `F` to 6
- **THEN** the file SHALL be written
- **AND** stdout SHALL be only `ok  <hash>  pkg/f.go  formatter <name>`

#### Scenario: Symbol patch of an already-high function may rise
- **GIVEN** function `F` has cyclomatic number 30 and `max_cc` is 30
- **WHEN** `patch --symbol F` raises `F` to 31
- **THEN** the file SHALL be written with only its ok line

#### Scenario: Create of a high Go function is refused
- **GIVEN** `pkg/` does not exist and `max_cc` is 30
- **WHEN** `create pkg/n.go --expect-absent --parents` gives a file with `A` at 2 and `B` at 31
- **THEN** neither `pkg/` nor `pkg/n.go` SHALL exist
- **AND** stdout SHALL be exactly `refused  pkg/n.go:B  cc new -> 31 crosses 30`
- **AND** the process SHALL exit 1 with no `Error:` line

#### Scenario: Create of an unparsed Go file or a non-Go file is written
- **WHEN** `create` gives a `.go` file go/parser rejects, or a `.py` file with forty branches
- **THEN** the file SHALL be written
- **AND** stdout SHALL be only the ok line

#### Scenario: Apply refuses the whole transaction
- **GIVEN** a manifest that edits `pkg/c.txt`, creates `pkg/b.go` with `B` at 31, and raises `F` in `pkg/a.go` from 29 to 30
- **WHEN** apply runs, with or without `--dry-run`
- **THEN** no file SHALL change and `pkg/b.go` SHALL NOT exist
- **AND** stdout SHALL be exactly `refused  pkg/a.go:F  cc 29 -> 30 crosses 30` then `refused  pkg/b.go:B  cc new -> 31 crosses 30`
- **AND** the process SHALL exit 1 with no `Error:` line

#### Scenario: Apply writes a rise from high, a removal, a non-Go file, and an unparsed Go file
- **GIVEN** a manifest that raises `F` from 30 to 33, removes `G` at 40, adds forty branches to a `.py` file, and edits a `.go` file go/parser rejects
- **WHEN** apply runs
- **THEN** every file SHALL be written
- **AND** stdout SHALL be one ok line per file and no `cc` or `refused` line
