## MODIFIED Requirements

### Requirement: Show command
`rgw-ast show <target>` SHALL print the body of a named symbol or a
`path:symbol` target when found under the workspace root. If multiple matches
exist, the tool SHALL list candidates and exit 1 unless the target disambiguates
by path. When the target has no `:` and names an existing file or directory,
show SHALL print that path's map outline instead of failing.

#### Scenario: Show known symbol
- **GIVEN** a Go function `Foo` exists in the workspace
- **WHEN** the user runs `rgw-ast show Foo` or `rgw-ast show path/to/file.go:Foo`
- **THEN** stdout SHALL contain the function body source

#### Scenario: Show a path
- **GIVEN** `AGENTS.md` exists at the workspace root
- **WHEN** the user runs `rgw-ast show AGENTS.md`
- **THEN** stdout SHALL contain the outline that `rgw-ast map AGENTS.md` prints
- **AND** the process SHALL exit 0

### Requirement: Search command
`rgw-ast search [flags] <query>` SHALL search workspace source (walk scope
applied) for literal substring matches and print one page of results
(`max_search_hits` from config). Each hit SHALL include path, line number, and
a content snippet truncated to at most 120 characters (with an ellipsis when
truncated). Flags SHALL be:
- `--path <dir>` and `--glob <pattern>` to scope the walk
- `--offset N` to skip the first N hits
- `--regex` to treat the query as a Go regular expression
- `-i` / `--ignore-case` for case-insensitive matching
- `--help`

Flags SHALL be parsed as flags, not as the query. `search --help` SHALL print
search help and exit 0 without scanning. When more hits exist than were
printed, stderr SHALL report the printed range, the total hit count, and the
next `--offset`.

#### Scenario: Search cap
- **GIVEN** more matches exist than `max_search_hits`
- **WHEN** search runs
- **THEN** at most `max_search_hits` hits SHALL be printed
- **AND** stderr SHALL report the total and the next `--offset`

#### Scenario: Search next page
- **GIVEN** 70 matches and `max_search_hits` 50
- **WHEN** search runs with `--offset 50`
- **THEN** the remaining 20 hits SHALL be printed

#### Scenario: Snippet truncation
- **GIVEN** a matching line longer than 120 characters
- **WHEN** search prints the hit
- **THEN** the content field SHALL be at most 120 characters plus an ellipsis
  indicator when truncated

#### Scenario: Search help is not a query
- **WHEN** the user runs `rgw-ast search --help`
- **THEN** the process SHALL print help and exit 0
- **AND** it MUST NOT treat `--help` as a search string

### Requirement: Explain path
`rgw-ast explain <path>` SHALL report whether the path is under the workspace
root, matches include/exclude, is under a nested git root, is binary, and
whether map/search would see it. For a directory it SHALL report how many
files under it are visible to map and search, and SHALL report the
directory's own skip reason when the walk scope skips it.

#### Scenario: Explain excluded path
- **WHEN** explain runs on a path under node_modules
- **THEN** output SHALL indicate exclusion or skip reason

#### Scenario: Explain a directory
- **GIVEN** `docs/` holds three included Markdown files
- **WHEN** explain runs on `docs`
- **THEN** output SHALL report `visible_files: 3`

## ADDED Requirements

### Requirement: Shared walk scope
`map`, `show`, `search`, `explain`, `callers`, `callees`, and `impact` SHALL
walk the same scope as `measure`: include/exclude globs, skipped directory
names, nested repositories (a `.git` directory or file), the root
`.gitignore` and `.git/info/exclude`, and directories whose own `.gitignore`
is the single rule `*`. Tool caches (`.ruff_cache`, `.mypy_cache`,
`.pytest_cache`, `.ast-bro`) SHALL be skipped by name. A path named directly
as a `map` argument SHALL still be mapped.

#### Scenario: Nested linked worktree is not indexed
- **GIVEN** a linked worktree (its `.git` is a file) inside the workspace root
- **AND** a function `F` defined once in the root and once in that worktree
- **WHEN** `rgw-ast callers F` runs at the root
- **THEN** `F` SHALL resolve to the root definition without ambiguity

#### Scenario: Ignored artifacts are not symbols
- **GIVEN** the root `.gitignore` ignores `tests/.artifacts/`
- **WHEN** `rgw-ast show` searches for a name defined only there
- **THEN** the symbol SHALL not be found

### Requirement: Ambiguous trace assessment
When `callers`, `callees`, or `impact` match more than one symbol, JSON output
SHALL include `assessment` with `conclusion` equal to `ambiguous`, and the
candidates, and the process SHALL exit 1.

#### Scenario: Ambiguous impact JSON
- **GIVEN** two functions named `F`
- **WHEN** `rgw-ast impact --json F` runs
- **THEN** JSON `assessment.conclusion` SHALL be `ambiguous`
