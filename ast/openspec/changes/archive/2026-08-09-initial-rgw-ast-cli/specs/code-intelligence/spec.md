## ADDED Requirements

### Requirement: Map command
`rgw-ast map [path]` SHALL list structural outline entries for a file or
directory under the workspace root. Paths are relative to the workspace root
or absolute within the root. When path is omitted, map SHALL outline the
workspace root subject to include/exclude and a reasonable entry cap from
config (`max_search_hits` or a dedicated map cap of at least 200 entries).

Each entry SHALL include at least path and a symbol or outline label when
available.

#### Scenario: Map a file
- **GIVEN** a source file exists under the workspace root
- **WHEN** the user runs `rgw-ast map <path>`
- **THEN** stdout SHALL list outline lines for that file
- **AND** the process SHALL exit 0

### Requirement: Show command
`rgw-ast show <target>` SHALL print the body of a named symbol or a
`path:symbol` target when found under the workspace root. If multiple matches
exist, the tool SHALL list candidates and exit 1 unless the target disambiguates
by path.

#### Scenario: Show known symbol
- **GIVEN** a Go function `Foo` exists in the workspace
- **WHEN** the user runs `rgw-ast show Foo` or `rgw-ast show path/to/file.go:Foo`
- **THEN** stdout SHALL contain the function body source

### Requirement: Search command
`rgw-ast search <query>` SHALL search workspace source (include/exclude
applied) for literal substring matches and print capped results
(`max_search_hits` from config). Each hit SHALL include path and line number.

#### Scenario: Search cap
- **GIVEN** more matches exist than `max_search_hits`
- **WHEN** search runs
- **THEN** at most `max_search_hits` hits SHALL be printed
- **AND** the tool MAY indicate that results were truncated
