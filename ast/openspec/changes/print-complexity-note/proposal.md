# Print a complexity note

## Why
Agents edit through `rgw-ast patch` and `rgw-ast apply`. Those commands succeed before Stop runs, so the agent does not learn that a function became new or worse until the turn is already over.

## What Changes
- After a successful `create`, `patch`, or `apply` that writes a source file, append the text of `cx note` for those files.
- Keep the command successful when `cx` is missing, times out, or exits non-zero.

## Capabilities
- cli-surface

## Non-goals
- Denying the write.
- Noting `append`, `delete`, or `move`.
- Adding a PostToolUse hook.

## Impact
Classification: tooling. Stdout of three verbs gains an optional note. Rollback is reverting `internal/cli/note.go` and the three call sites. Exit codes stay the same.
