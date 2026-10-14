## ADDED Requirements

### Requirement: Hash-guarded move
`rgw-ast move <src> <dst> --expect-hash <sha256> [--parents]` SHALL move one
regular file inside the workspace root when its hash matches and `dst` does
not exist. Both paths SHALL resolve inside the root. Without `--parents` the
destination directory SHALL already exist. When `src` is tracked by git the
move SHALL use `git mv` so the rename is staged; otherwise it SHALL rename the
file. On success stdout SHALL report the hash and both paths.

#### Scenario: Move a tracked file
- **GIVEN** `a.go` is tracked with hash H and `b.go` is absent
- **WHEN** the user runs `rgw-ast move a.go b.go --expect-hash H`
- **THEN** `b.go` SHALL exist with hash H and `a.go` SHALL not
- **AND** git SHALL record the rename as staged

#### Scenario: Existing destination is refused
- **GIVEN** `b.go` exists
- **WHEN** the user runs `rgw-ast move a.go b.go --expect-hash H`
- **THEN** the process SHALL exit 1 and neither file SHALL change

#### Scenario: Stale hash is refused
- **GIVEN** `a.go` does not have hash H
- **WHEN** the user runs `rgw-ast move a.go b.go --expect-hash H`
- **THEN** the process SHALL exit 1 and `a.go` SHALL remain
