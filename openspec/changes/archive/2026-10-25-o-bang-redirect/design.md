## Context

The colon line runs `mv`, `rm`, `cp`, `touch`, and `mkdir`. A trimmed line that starts with `!` runs the rest in the shell. `touch` creates a missing file and does not empty an existing one.

## Goals / Non-Goals

- Goals: `!> path` and `!>path` create that file or empty it, then select it. Quotes around one path work. The terminal stays with the navigator.
- Non-Goals: `>>`, a redirect with extra words, and every other `!` line stay shell commands. Globs stay unexpanded.

## Decisions

- A bare `>` and one path is a file redirect. `os.OpenFile` with create and truncate writes it in the listed directory. An absolute path is used as given.
- `>>` and more than one word after `>` are not this redirect. They keep the shell handoff.
- `touch` is unchanged, so existing text stays.

## Risks / Trade-offs

- A path whose parent is missing fails, and the listing stays as it was.
- Redirecting onto a directory fails and leaves the directory in place.

## Migration Plan

Type `!> hello.txt` on the colon line. `touch hello.txt` still does not empty the file.

## Open Questions

None.
