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

#### Scenario: Map a file
- **GIVEN** a source file exists under the workspace root
- **WHEN** the user runs `rgw-ast map <path>`
- **THEN** stdout SHALL list outline lines for that file
- **AND** the process SHALL exit 0

#### Scenario: Map bash functions
- **GIVEN** a shell file defining `foo()`
- **WHEN** map runs on that file
- **THEN** output SHALL include a function entry for `foo`

### Requirement: Search command
`rgw-ast search [flags] <query>` SHALL search workspace source (include/exclude
applied) for literal substring matches and print capped results
(`max_search_hits` from config). Each hit SHALL include path, line number, and
a content snippet truncated to at most 120 characters (with an ellipsis when
truncated). Flags `--path`, `--glob`, and `--help` SHALL be parsed as flags,
not as the query. `search --help` SHALL print search help and exit 0 without
scanning.

#### Scenario: Search cap
- **GIVEN** more matches exist than `max_search_hits`
- **WHEN** search runs
- **THEN** at most `max_search_hits` hits SHALL be printed
- **AND** the tool MAY indicate that results were truncated

#### Scenario: Snippet truncation
- **GIVEN** a matching line longer than 120 characters
- **WHEN** search prints the hit
- **THEN** the content field SHALL be at most 120 characters plus an ellipsis
  indicator when truncated

#### Scenario: Search help is not a query
- **WHEN** the user runs `rgw-ast search --help`
- **THEN** the process SHALL print help and exit 0
- **AND** it MUST NOT treat `--help` as a search string

## ADDED Requirements

### Requirement: Explain path
`rgw-ast explain <path>` SHALL report whether the path is under the workspace
root, matches include/exclude, is under a nested git root, is binary, and
whether map/search would see it.

#### Scenario: Explain excluded path
- **WHEN** explain runs on a path under node_modules
- **THEN** output SHALL indicate exclusion or skip reason
