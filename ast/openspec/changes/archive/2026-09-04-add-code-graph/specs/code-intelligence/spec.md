## MODIFIED Requirements

### Requirement: Map command
`rgw-ast map [path]` SHALL list structural outline entries for a file or
directory under the workspace root. Paths are relative to the workspace root
or absolute within the root. When path is omitted, map SHALL outline the
workspace root subject to include/exclude and a reasonable entry cap from
config (`max_search_hits` or a dedicated map cap of at least 200 entries).

Each entry SHALL include at least path and a symbol or outline label when
available. For `.sh`/Bash files it SHALL list function definitions when
detectable. Direct path arguments SHALL be mapped even when the path would not
match include globs for directory walks.

When mapping a directory, entries SHALL be ordered by descending import-graph
degree of the file's Go package directory (then path, then line). Files not
in the Go import graph keep their relative order after ranked Go files, or
interleaved at degree 0.

#### Scenario: Map a file
- **GIVEN** a source file exists under the workspace root
- **WHEN** the user runs `rgw-ast map <path>`
- **THEN** stdout SHALL list outline lines for that file
- **AND** the process SHALL exit 0

#### Scenario: Map bash functions
- **GIVEN** a shell file defining `foo()`
- **WHEN** map runs on that file
- **THEN** output SHALL include a function entry for `foo`

#### Scenario: Ranked directory map
- **GIVEN** two Go package dirs where A is imported by more workspace
  packages than B
- **WHEN** map runs on a directory containing both
- **THEN** outline lines for files in A MUST appear before those in B
