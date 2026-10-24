## Why

The navigator can create, cut, copy, and paste a file or directory. Removing one still needs a typed `rm` command, and that command refuses a directory that contains anything.

## What Changes

- `dd` deletes the current file or directory in `c3po tui o`.
- The first `d` arms the chord and shows `d`. Esc cancels it. Any other key clears the arm and runs as a normal motion.
- A directory is removed with its contents. A symlink is removed without following it.

## Impact

- Spec: `openspec/specs/o/spec.md`
- Code: `o/internal/app`, `o/help.go`
