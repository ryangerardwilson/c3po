# exec-generator Specification

## Purpose
Run trusted scaffolding generators from the global allowlist and report every file they create or modify.
## Requirements
### Requirement: Trusted generator exec
`c3po ast exec -- <command...>` SHALL run an allowlisted generator under the
workspace root, capture Git working-tree and ignored-path snapshots before and
after, and report only content changes attributable to the generator, with hashes
when available. `rgw-ast exec` SHALL do the same work. Pre-existing unchanged paths SHALL NOT be reported as changes.
Commands not structurally matching `generators.allow` from argv[0] SHALL be
rejected without execution.

#### Scenario: Reject unlisted command
- **WHEN** exec runs `-- rm -rf x`
- **THEN** the process SHALL fail without running rm


#### Scenario: Reject allowlist phrase smuggling
- **WHEN** exec runs `-- bash -c <script> "npm exec -- openspec"`
- **THEN** the process SHALL fail without running bash

#### Scenario: Audit ignored content deltas
- **GIVEN** an ignored path exists before the generator runs
- **WHEN** the generator leaves that path unchanged and creates another ignored path
- **THEN** only the newly created ignored path SHALL appear in `ignored_observed`

#### Scenario: Audit modified ignored content
- **GIVEN** an ignored file exists before the generator runs
- **WHEN** the generator changes its content
- **THEN** `ignored_observed` SHALL report it as `modified_ignored` with its resulting hash
