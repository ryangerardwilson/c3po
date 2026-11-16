## MODIFIED Requirements

### Requirement: Cyclomatic gate and cc lines on plain Go patch
A plain patch SHALL, after the hash matches and the replacements are applied in memory and before the file is written, score every function in the old and the new bytes when the file name ends in `.go`, `.ts`, `.tsx`, or `.py`. `patch --symbol`, `create`, `apply`, and `append` SHALL be gated by the same refusal rule. A `move` or `delete` SHALL NOT be scored. JavaScript, Rust, and any other extension SHALL NOT be scored.

A Go declaration SHALL be named as cyclotree names it: the receiver type with a leading pointer star removed, `.`, and the function name; a function with no receiver is its name. Go function literals SHALL NOT be scored on their own and SHALL NOT add their body to the enclosing declaration. TypeScript and Python functions SHALL use the name cyclotree reports. The score SHALL be the cyclomatic number cyclotree reports.

Go bytes are a clean parse when go/parser accepts them and no two scored functions share a name. When either side is not a clean parse, nothing SHALL be written, stdout SHALL be exactly `refused  <relpath>  unparsed`, stderr SHALL NOT hold an `Error:` line, and the process SHALL exit 1.

When both sides are clean, the patch SHALL be refused when any function moves from below `max_cc` to `max_cc` or above, or a function only in the new bytes is at `max_cc` or above. A function already at or above `max_cc` MAY rise. A removal SHALL NOT refuse.

When not refused, a plain patch SHALL be written and stdout SHALL be the ok line followed by one cc line per changed function. `create`, `apply`, and `append` SHALL NOT print a cc line on success. When refused, nothing SHALL be written, stdout SHALL hold one `refused  <relpath>:<Name>  cc <old> -> <new> crosses <max_cc>` or `refused  <relpath>:<Name>  cc new -> <score> crosses <max_cc>` line per refusing function, sorted by path and then by name, and the process SHALL exit 1 with no `Error:` line.

A hash mismatch SHALL print no cc line and no refused line.

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
- **WHEN** a plain patch of `f.txt` adds many branches, or of `f.py` adds branches that are not inside a function
- **THEN** the file SHALL be written
- **AND** stdout SHALL be only the ok line

#### Scenario: A text file has no cc line
- **WHEN** a plain patch of `f.txt` adds many branches
- **THEN** the file SHALL be written
- **AND** stdout SHALL be only the ok line

#### Scenario: A Python function over the line is refused
- **WHEN** a create or patch adds a Python function at max_cc or above
- **THEN** the file SHALL NOT contain that function
- **AND** stdout SHALL name the function and say it crosses max_cc

#### Scenario: A TypeScript function over the line is refused
- **WHEN** a create or patch adds a TypeScript function at max_cc or above
- **THEN** the file SHALL NOT contain that function
- **AND** stdout SHALL name the function and say it crosses max_cc

#### Scenario: Unparsed Go file is written
- **GIVEN** `f.go` is rejected by go/parser
- **WHEN** a plain patch matches and applies
- **THEN** the file SHALL NOT be written
- **AND** stdout SHALL be exactly `refused  f.go  unparsed`
- **AND** the process SHALL exit 1

#### Scenario: Unparsed Go is refused
- **GIVEN** `f.go` is rejected by go/parser
- **WHEN** a plain patch matches and applies
- **THEN** the file SHALL NOT be written
- **AND** stdout SHALL be exactly `refused  f.go  unparsed`
- **AND** the process SHALL exit 1

#### Scenario: Hash mismatch prints no cc line
- **WHEN** a plain patch of a Go file gives a stale `--expect-hash`
- **THEN** nothing SHALL be written
- **AND** stdout SHALL hold no cc or refused line
- **AND** the process SHALL exit 1 with the hash mismatch error

#### Scenario: Append over the line is refused
- **WHEN** append adds a Go, TypeScript, or Python function at max_cc or above
- **THEN** the file SHALL NOT change
- **AND** stdout SHALL name the function and say it crosses max_cc

### Requirement: Cyclomatic gate on symbol patch, create, and apply
`patch --symbol`, `create`, `apply`, and `append` SHALL score the functions of every `.go`, `.ts`, `.tsx`, and `.py` file they would write, with the names and the counter of the plain patch gate, on the bytes that would be written. The old bytes of an edit SHALL be the file before the write. A created file SHALL have no old functions. A `move` or `delete` SHALL NOT be scored. Files of any other extension SHALL NOT be scored and SHALL NOT be refused for cyclomatic complexity.

The write SHALL be refused when any function moves from below `max_cc` to `max_cc` or above, or a function only in the new bytes is at `max_cc` or above. A function already at or above `max_cc` MAY rise. A removal SHALL NOT refuse. When the old or the new bytes are not a clean parse, the write SHALL be refused, nothing SHALL be written, and stdout SHALL be `refused  <relpath>  unparsed`.

When refused, nothing SHALL be written: no file, no created directory, and no other file of the same `apply` transaction. Stdout SHALL hold one refused line per refusing function, sorted by path and then by name. Stderr SHALL NOT hold an `Error:` line, and the process SHALL exit 1. `apply --dry-run` SHALL refuse the same way.

When not refused, output SHALL be the ok line and no `cc` line. A hash mismatch SHALL fail with no refused line.

#### Scenario: Symbol patch crossing max_cc is refused
- **GIVEN** function `F` in `pkg/f.go` has cyclomatic number 29 and `max_cc` is 30
- **WHEN** `patch pkg/f.go --symbol F` raises `F` to 30
- **THEN** the file SHALL NOT be written
- **AND** stdout SHALL be exactly `refused  pkg/f.go:F  cc 29 -> 30 crosses 30`
- **AND** the process SHALL exit 1

#### Scenario: Symbol patch under max_cc prints no cc line
- **GIVEN** function `F` in `pkg/f.go` has cyclomatic number 5
- **WHEN** `patch --symbol F` raises `F` to 6
- **THEN** the file SHALL be written
- **AND** stdout SHALL be only the ok line

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
- **WHEN** `create` gives a `.go` file go/parser rejects
- **THEN** the file SHALL NOT be written
- **AND** stdout SHALL say unparsed
- **WHEN** `create` gives a `.txt` file or a `.js` file with many branches
- **THEN** the file SHALL be written
- **AND** stdout SHALL be only the ok line

#### Scenario: Create of unparsed Go is refused
- **WHEN** `create` gives a `.go` file go/parser rejects
- **THEN** the file SHALL NOT be written
- **AND** stdout SHALL say unparsed

#### Scenario: Create of a non-scored file is written
- **WHEN** `create` gives a `.txt` file or a `.js` file with many branches
- **THEN** the file SHALL be written
- **AND** stdout SHALL be only the ok line

#### Scenario: Apply refuses the whole transaction
- **GIVEN** a manifest that edits `pkg/c.txt`, creates `pkg/b.go` with `B` at 31, and raises `F` in `pkg/a.go` from 29 to 30
- **WHEN** apply runs, with or without `--dry-run`
- **THEN** no file SHALL change and `pkg/b.go` SHALL NOT exist
- **AND** stdout SHALL be exactly `refused  pkg/a.go:F  cc 29 -> 30 crosses 30` then `refused  pkg/b.go:B  cc new -> 31 crosses 30`
- **AND** the process SHALL exit 1 with no `Error:` line

#### Scenario: Apply writes a rise from high, a removal, a non-Go file, and an unparsed Go file
- **GIVEN** a manifest that raises `F` from 30 to 33, removes `G` at 40, edits a `.txt` file, and edits a `.go` file go/parser rejects
- **WHEN** apply runs
- **THEN** no file SHALL change
- **AND** stdout SHALL be `refused  <relpath>  unparsed`
- **AND** the process SHALL exit 1
- **WHEN** the same manifest omits the unparsed Go file
- **THEN** the rise, the removal, and the `.txt` file SHALL be written
- **AND** stdout SHALL be one ok line per file and no `cc` or `refused` line
