## Why

Enforced workspaces can create and patch files safely but cannot remove stale
files without bypassing rgw-ast through `rm`, `git rm`, or a direct deletion
tool. This leaves the mutation boundary incomplete precisely when a cleanup or
migration requires intentional deletion.

## What Changes

- Add a `delete` command that removes exactly one regular file only when its
  current SHA-256 matches `--expect-hash`.
- Add `--prune-empty` to remove empty ancestor directories after the guarded
  file deletion, stopping at the workspace root and never deleting non-empty
  directories.
- Keep direct directory, recursive, symlink, root, and outside-workspace
  deletion unsupported.
- Document the command in help, README, and the managed AGENTS boundary.
- Add file-layer, CLI, and hook-facing regression coverage.

## Capabilities

### New Capabilities

- `file-delete`: Hash-guarded single-file deletion and bounded empty-directory
  pruning inside the resolved workspace.

### Modified Capabilities

- `cli-surface`: Add `delete` to the flat top-level CLI grammar.

## Impact

The public CLI, `internal/cli`, `internal/files`, help text, embedded AGENTS
guidance, README, and tests change. No configuration schema or external
dependency changes. Existing commands remain compatible.

## Non-goals

- Recursive or glob-based deletion.
- Deleting a directory directly, even when empty.
- Deleting symlinks, special files, nested repositories, or paths outside the
  selected workspace.
- Trash/recovery management or source-control staging.
