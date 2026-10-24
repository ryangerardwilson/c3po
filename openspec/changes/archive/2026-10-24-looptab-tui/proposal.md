## Why

Looptab can run one configured job immediately with `looptab now`, but choosing that job is a typed index. The other c3po tools that pick from a short list do it from a screen.

## What Changes

- `c3po tui looptab` lists the configured Looptab jobs.
- `j` and `k` move. Enter runs the selected job once through `looptab now <index>`.
- Shortcut text stays behind `?`.
- `c3po looptab` is not a command.

## Impact

- Specs: `openspec/specs/looptab/spec.md`, `openspec/specs/scout/spec.md`, `openspec/specs/tui/spec.md`
- Code: `looptab/`, `internal/scout`
