## Context

`:` opens a colon line that shows `: ` and refuses a leading bang. `!` opens a different line that shows `! ` and a hint. The line is append-only.

## Goals / Non-Goals

- Goals: vim adjacency (`:` against the text, `:!` for a shell), and Emacs editing keys on that one line.
- Non-Goals: Emacs keys in the filter or while browsing. A kill ring. Changing the file-command allowlist.

## Decisions

- One command line. `!` prefills `!` and places the cursor after it. The `shelling` flag goes away.
- The drawn line is `:` plus the text, with `|` at the cursor. An empty line is `:|`. There is no hint.
- Enter trims the line. A leading `!` runs the rest through the existing shell handoff. Anything else stays on mv, rm, cp, touch, and mkdir.
- Words are whitespace-separated. Ctrl-w deletes the spaces immediately before the cursor and the word before them. Ctrl-d at the end of the line does nothing and does not quit. Alt chords that are not editing keys do not insert a letter.
- The shortcut overlay stays 24 lines. The `:` and `!` descriptions change in place.

## Risks / Trade-offs

- `:!` can run any shell command in the listed directory. That is the requested vim behavior. A line that does not start with `!` cannot.
