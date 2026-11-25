## Why

`c3po gates` prints a hardcoded statute for an agent to judge work against. The operator now keeps those rules as r2d2 skills, so the command, its statute, and its tally sheet are unused weight.

## What Changes

- **BREAKING** `c3po gates` is not a command. It exits 2 as an unknown command, and `c3po` and `c3po help` no longer list it.
- The `gates` package, the architecture statute, and the gates help are deleted.
- The usage catalog no longer has a `gates` feature. `c3po gates`, including `c3po gates --help` and `c3po gates version`, records `unknown`.
- `dirt` is listed between `deport` and `gcal`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `gates`: the statute requirements are removed. One requirement remains: `c3po gates` is an unknown command.
- `scout`: `gates` is not an app, and `dirt` is listed between `deport` and `gcal`.

## Impact

- Deletes `gates/` and drops the command from `internal/scout`, `internal/usage`, the README, and `AGENTS.md`.
- `openspec/specs/gates/spec.md` states only that the command is unknown. `openspec/specs/scout/spec.md` stops listing the command.
- No database migration. The version stays 0.5.0.
- `c3po gates` is an unknown command and exits 2. The rules live in r2d2 skills.
