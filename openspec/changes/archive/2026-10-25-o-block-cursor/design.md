## Context

The command line inserts `|` between characters. The terminal cursor is hidden while the navigator paints, and shown again for a shell command and on exit. Nothing sets its shape.

## Goals / Non-Goals

- Goals: a block cursor on the command line, on the filter, and for the terminal cursor while the navigator owns the screen.
- Non-Goals: changing the listing `>` marker. Changing the cursor on other c3po screens.

## Decisions

- A block is reverse video on one cell. At the end of the text the cell is a space. On a character, that character is the cell.
- The navigator sends a steady block (`CSI 2 q`) when it takes the terminal, and the terminal default (`CSI 0 q`) when it gives the terminal back, including around a shell command.
- The listing cursor marker stays `>`.

## Risks / Trade-offs

- A terminal that ignores cursor-shape sequences keeps its configured shape. The drawn command and filter cursors are still blocks.
