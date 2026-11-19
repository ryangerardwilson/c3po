## Why

An agent that runs `git worktree add` leaves a linked worktree nobody marked, so `c3po mess tidy` can never remove it. The hook allows the command today because `worktree` is not a writing git subcommand and the new path is outside every current worktree.

## What Changes

- The hook denies `git worktree add`, including `git -C <dir> worktree add` and the same command inside `sh -c` or `bash -c`, when the effective directory is inside a git repository. Enforcement does not matter. The reason says `use c3po mess tidy add`, and also says to run `c3po mess tidy` first when the repository already has a clean marked worktree.
- The hook denies `c3po mess tidy add` only while a clean marked worktree exists, and the reason names `c3po mess tidy`.
- `git worktree list`, `git worktree remove`, `c3po mess`, and `c3po mess tidy` stay allowed.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-hook`: linked worktrees are created through `c3po mess tidy add`.

## Impact

- Spec: `ast/openspec/specs/agent-hook/spec.md`
- Code: `ast/internal/hook`
- The clean check runs only when a matching command appears, and it may call `git ls-remote` for marked worktrees.
- The AGENTS block is unchanged.
