## Context

`dispatch` treats the first argument as a command name. `open` is the command that stats paths. Empty arguments already start the navigator.

## Goals / Non-Goals

**Goals:**

- `c3po tui o <path>` is the form for a directory or files.
- Help and the usage line show that form.

**Non-Goals:**

- Keeping `open` as a hidden alias.
- Changing reveal, pick, save, or config.

## Decisions

A first argument that is not `reveal`, `pick`, `save`, or `config` is a path list. `help` and `version` are still handled before dispatch. One directory starts the navigator. Several targets must all be files. A missing path exits 1 with `Path not found`.

## Risks / Trade-offs

A file or directory whose name is `reveal`, `pick`, `save`, `config`, `help`, or `version` is still selected by that command name. A path named `open` is now a path.

## Migration Plan

Use `c3po tui o <path>`. The shell function `o` already forwards its arguments.

## Open Questions

None.
