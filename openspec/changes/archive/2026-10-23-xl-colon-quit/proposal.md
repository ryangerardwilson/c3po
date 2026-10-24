## Why

The spreadsheet leaves through `q`, ctrl-c, `:q`, and a yes-or-no prompt. Those paths save or discard beside the command line.

## What Changes

- `:w` is the save command. `:wq` saves and quits. `:q!` quits and discards.
- `q`, ctrl-c, and `:q` no longer leave the sheet. The save-or-discard prompt is gone.

## Impact

- Spec: `openspec/specs/xl/spec.md`
- Code: `xl/internal/tui`, `xl/README.md`
