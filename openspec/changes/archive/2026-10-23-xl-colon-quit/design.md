## Context

`:w` saves and asks for a path when the workbook has none. `:wq` saves and quits. `:q`, `q`, and ctrl-c call a prompt: `y` saves and quits, `n` discards, Esc stays.

## Goals / Non-Goals

- Save only with `:w` or `:wq`.
- Quit and keep the write only with `:wq`.
- Quit and discard only with `:q!`.
- Keep the path prompt that `:w` and `:wq` open when the workbook has no path.
- Leave cell editing, search, and rename as they are.

## Decisions

- `q` and ctrl-c in normal mode do nothing. ctrl-c while editing commits the cell and returns to normal, the same as Esc.
- `:q` is an unknown command. `:q!` quits without writing.
- The yes-or-no quit mode is removed.

## Risks / Trade-offs

- A modified sheet stays open until `:wq` or `:q!`.
