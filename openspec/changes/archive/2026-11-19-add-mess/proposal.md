## Why

Agents create linked worktrees and leave them behind. Nothing tells the next agent how messy the repository is, which worktrees it may remove, or that it should stop making new ones until the finished ones are gone. Removing them by hand risks unpushed work.

## What Changes

- `c3po mess` prints a report of the current repository's linked worktrees: clean, keep, other, and blocked. It deletes nothing and exits 0.
- `c3po mess tidy` removes only the worktrees the report calls clean, with `git worktree remove` and no `--force`.
- `c3po mess tidy add <path> [-b <branch>]` creates one linked worktree and writes a `c3po-tidy` marker in its git admin directory with the session id, the UTC time, and the path. It refuses while a clean entry exists.
- `c3po mess tidy --session <id>` removes only clean worktrees with that marker session. `c3po mess tidy --hook` reads a SessionEnd JSON object and runs the session tidy from its cwd. The hook path always exits 0.
- `mess` is an app in the c3po command list and help. There is no mess screen.

## Capabilities

### New Capabilities

- `mess`: report linked worktrees, tidy the clean marked ones, and create marked worktrees.

### Modified Capabilities

- `scout`: `mess` is an app in the command list and the help list.

## Impact

- Specs: `openspec/specs/mess/spec.md`, `openspec/specs/scout/spec.md`
- Code: `mess/`, `internal/scout`, `internal/usage`
- The ast hook denies a bare `git worktree add`; that is the ast change `deny-worktree-add`.
- Nothing is stashed, reset, force-removed, pruned, or deleted from a branch. `~/Work` is not scanned.
