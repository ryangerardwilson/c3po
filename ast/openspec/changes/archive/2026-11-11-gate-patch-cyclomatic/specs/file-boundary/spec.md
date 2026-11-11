## ADDED Requirements

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
