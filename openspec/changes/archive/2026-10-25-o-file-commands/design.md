## Context

`:` opens a command line. An empty line adds a hint, and a line that starts with `!` runs a shell. The allowlist is touch, mkdir, and rm, and mkdir may take `-p`.

## Goals / Non-Goals

- Goals: a bare colon line, and only mv, rm, `rm -rf`, cp, touch, and mkdir.
- Non-Goals: changing the `!` key. Allowing other flags, including `mkdir -p`, `rm -r`, and `cp -r`.

## Decisions

- The empty colon line is `: ` plus the cursor. The shell line still names itself when empty.
- `rm` accepts `-rf` as one argument. cp and mv need two paths. The other commands need one.
- `colonShell` is removed, so a colon line cannot hand work to the shell.

## Risks / Trade-offs

- `rm -rf` deletes a directory tree in the listed directory. That is the requested command.
