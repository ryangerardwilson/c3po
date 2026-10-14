## ADDED Requirements

### Requirement: Trusted generator exec
`rgw-ast exec -- <command...>` SHALL run an allowlisted generator under the
workspace root, capture git porcelain before and after, and report changed
paths with hashes when available. Commands not matching `generators.allow`
SHALL be rejected without execution.

#### Scenario: Reject unlisted command
- **WHEN** exec runs `-- rm -rf x`
- **THEN** the process SHALL fail without running rm
