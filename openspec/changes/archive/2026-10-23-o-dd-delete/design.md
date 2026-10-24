## Context

Drive and deport already use `dd` as an armed chord. The navigator lists one directory and acts on the current row with `x`, `y`, and `p`. `!` can run `rm` on one name and refuses other flags.

## Goals / Non-Goals

- Delete the current file or directory with `dd`.
- Arm on the first `d`, show `d`, cancel with Esc, and let any other key run normally.
- Remove a directory tree. Remove a symlink itself.
- Leave `!` and its `rm` allowlist unchanged.
- Leave the clipboard, cut, and paste behavior unchanged except that a stage pointing at a deleted path is cleared.

## Decisions

- The chord matches Drive and deport: the first `d` sets the arm and the status `d`. The second `d` deletes. Esc clears the arm. Any other key clears the arm and is handled as itself, including `q`.
- `d` typed into the filter or the `!` prompt stays text.
- Deletion uses `os.RemoveAll` on the current row. That removes a file, a directory and its contents, and a symlink without removing the symlink target.
- An empty listing reports `nothing to delete`.
- After a successful delete the listing reloads and the cursor stays at the same index. The status is `deleted <name>`.
- When the staged path is the deleted path, or is inside a deleted directory, the stage is cleared.

## Risks / Trade-offs

- A directory delete is permanent after the second `d`. The arm is the confirmation.
- The footer line grows by `dd delete`.
