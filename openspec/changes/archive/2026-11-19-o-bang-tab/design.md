## Context

`!` opens a `:!` line. Enter runs the rest in the shell with the listed directory as its working directory. Tab is not bound there.

## Goals / Non-Goals

- Goals: Tab completes the word at the cursor the way bash does for an ordinary command and for paths.
- Non-Goals: programmable completion, usernames after `~user`, and menu cycling.

## Decisions

- The first word is a command unless it contains `/` or starts with `~`. Later words are paths. Paths resolve from the listed directory. `~/` stays typed as `~/`.
- Dot names match only when the typed base starts with `.`. `.` and `..` are not matches.
- One directory gains `/` and no space. One file or command gains a space, unless a space is already next. Several matches insert the longest common prefix. When the word does not change, the prompt line shows the names and not the command text.
- A character the shell would split or expand is backslash-escaped in an unquoted word. A word that already starts with a quote keeps that quote.
- Tab on a line that does not start with `!` does nothing. Any other key clears the match list.

## Risks / Trade-offs

- An empty command word lists every executable on `PATH`. The prompt line shows only what fits on one row.

## Migration Plan

Press Tab on a `:!` line. Existing Enter behavior is unchanged.

## Open Questions

None.
