## Context

The frame is written inline and is only as long as the text the view returns. `visibleWindow` reserves `height-6` rows and also forces at least 8, so a short terminal clips the title. There is no command prompt.

## Goals / Non-Goals

**Goals:**

- The painted view is one line per terminal row.
- `!` runs `touch`, `mkdir`, and `rm` in the listed directory.

**Non-Goals:**

- A general shell.
- Flags other than `mkdir -p`.

## Decisions

The view is title, one prompt line, the listing, the status, and the key hint. Listing rows are `height - 4`, and short listings pad with blank rows so the hint stays on the last line. The renderer also pads a short frame up to the terminal height.

`!` uses the same input line as the filter. Enter runs the words with `exec.Command` and `Dir` set to the listed directory. The name must be `touch`, `mkdir`, or `rm`. A flag is refused except `mkdir -p`. The listing reloads and the last path is selected.

## Risks / Trade-offs

A name that needs a shell quote is typed with quotes. The command does not expand globs. Paths may point outside the listed directory when the operator types them that way.

## Migration Plan

`!` is a new key. Existing keys stay.

## Open Questions

None.
