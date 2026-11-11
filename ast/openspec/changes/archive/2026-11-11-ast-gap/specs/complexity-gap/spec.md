## ADDED Requirements

### Requirement: Gap report
`rgw-ast gap [--json] [--check]` SHALL list every scored function whose
cyclomatic number is at or above `max_cc` (config, default 30). It SHALL walk
the workspace root with the same scope as `measure`, `map`, and trace:
include and exclude globs, the root `.gitignore`, nested git repositories,
and directories that ignore themselves. `--root` SHALL select the root as it
does for `measure`.

Each walked file SHALL be scored with the cyclotree scorer when its extension
is one cyclotree scores: `.go` as `go`, `.py` as `python`, `.rs` as `rust`,
`.ts` and `.tsx` as `typescript`, `.js`, `.jsx`, `.mjs`, and `.cjs` as
`javascript`. Other files SHALL be absent from the report and SHALL NOT be an
error. A function whose cyclomatic number is 0 or less SHALL be skipped. A Go
function's number SHALL equal the number the plain patch gate and gocyclo
give it.

Functions SHALL sort by cyclomatic number descending, then by path, then by
name. Paths SHALL be slash paths relative to the root.

Text output SHALL be `gap  <absolute root>`, `max_cc  <n>`, and
`over  <count>`, then one line per function,
`<cyclomatic>  <path>:<name>  <language>`. When `over` is 0 there SHALL be no
function line.

`--json` SHALL print one object
`{"root","max_cc","over","functions":[{"path","name","language","line","cyclomatic"}]}`
in that sort order; `functions` SHALL be an empty array, not null, when
`over` is 0.

Without `--check` the process SHALL exit 0 even when `over` is not 0. With
`--check` it SHALL print the same report and exit 1 when `over` is not 0,
otherwise 0, with no `Error:` line. A walk or read failure SHALL print an
`Error:` line on stderr and exit 1. An unknown flag or an operand SHALL exit
2. `gap` SHALL only report; it SHALL NOT write a file.

#### Scenario: Text lists functions at or above max_cc
- **GIVEN** `max_cc` is 30, `a.go` has `Hi` at 30 and `Lo` at 29, and `b.py` has `g` at 31
- **WHEN** `rgw-ast gap` runs
- **THEN** stdout SHALL be `gap  <root>`, `max_cc  30`, `over  2`, `31  b.py:g  python`, `30  a.go:Hi  go`
- **AND** the process SHALL exit 0

#### Scenario: A Go function at exactly max_cc is listed
- **GIVEN** Go function `Hi` has cyclomatic number 30 and `max_cc` is 30
- **WHEN** gap runs
- **THEN** `Hi` SHALL be listed with 30, the number the patch gate gives it

#### Scenario: A function at max_cc minus one is omitted
- **GIVEN** Go function `Lo` has cyclomatic number 29 and `max_cc` is 30
- **WHEN** gap runs
- **THEN** no line SHALL name `Lo`

#### Scenario: Another language is listed with its language name
- **GIVEN** Python function `g` in `b.py` has cyclomatic number 31
- **WHEN** gap runs
- **THEN** its line SHALL end with `python`

#### Scenario: Skipped directories are not scored
- **GIVEN** `vendor/v.go` and a gitignored `gen/g.go` each hold a function above `max_cc`
- **WHEN** gap runs
- **THEN** neither SHALL be listed

#### Scenario: JSON
- **WHEN** `rgw-ast gap --json` runs on the same tree
- **THEN** stdout SHALL be an object with `root`, `max_cc` 30, `over` 2, and `functions` holding `b.py` `g` then `a.go` `Hi`, each with `path`, `name`, `language`, `line`, and `cyclomatic`
- **AND** the process SHALL exit 0

#### Scenario: Check fails on debt
- **GIVEN** at least one function is at or above `max_cc`
- **WHEN** `rgw-ast gap --check` runs
- **THEN** stdout SHALL be the same report
- **AND** stderr SHALL be empty
- **AND** the process SHALL exit 1

#### Scenario: Empty debt
- **GIVEN** no scored function is at or above `max_cc`
- **WHEN** `rgw-ast gap` or `rgw-ast gap --check` runs
- **THEN** stdout SHALL be only the `gap`, `max_cc`, and `over  0` lines
- **AND** the process SHALL exit 0
- **AND** `--json` SHALL give `"functions": []`

#### Scenario: Usage errors
- **WHEN** `rgw-ast gap extra` or `rgw-ast gap --bogus` runs
- **THEN** stderr SHALL hold an `Error:` line
- **AND** the process SHALL exit 2
