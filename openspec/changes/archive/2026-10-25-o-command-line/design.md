## Context

The screen is title, prompt line, listing, status. `!` draws `! <input>` on the prompt line and puts `touch, mkdir, or rm` on the status line.

## Goals / Non-Goals

**Goals:**

- The command being typed occupies the last row.
- The listing stays on screen while the command line is open.

**Non-Goals:**

- Moving the `/` filter off the prompt line.
- Allowing commands other than `touch`, `mkdir`, and `rm`.

## Decisions

While `commanding` is set, the last row is the command line and the status text is not drawn. The line is `! `, the input, and a `|` cursor. With an empty input it also names `touch, mkdir, or rm`. The prompt line under the title keeps the clipboard label or the filter, and does not repeat the command. Enter and Esc close the command line, and the status row returns.

## Risks / Trade-offs

A long command is cut from the left so the cursor stays on the row.

## Migration Plan

`!`, Enter, and Esc behave as they do now. Only the row changes.

## Open Questions

None.
