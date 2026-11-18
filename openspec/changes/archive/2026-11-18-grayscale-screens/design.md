# Grayscale screens

## Context

r2d2 hosts the c3po screens in-process. Those screens encode magnitude and state with hue: yellow and cyan on `l` and `o`, yellow on cyclotree, and cyan, yellow, and red on `xl`. The list screens mark the active row bright green and an error bright red.

## Goals

- Every painted channel is equal, so the color is black, white, or gray.
- A larger file, a larger binary, and a stronger state stay brighter.
- Bold, dim, and reverse stay available.
- The cursor on `xl` stays visible against the focused cell.

## Non-goals

- Recoloring `c3po gmail` or `c3po gcal`.
- Changing the Omarchy terminal theme.
- Changing what a row means, or which key does what.

## Decisions

- Directories on `l` stay white. Text files and binaries use the existing log scale on all three channels. Tree glyphs stay mid gray.
- Cyclotree paints the frame white. Root rows stay bold and leaves stay dim.
- On `o`, unselected text keeps the 70 to 255 scale as gray. A selected text row sits on dark gray, so its foreground is at least 180. That keeps a short file readable and still lets a long file go white. Status is light gray. Muted text stays index 241.
- On `xl`, the focused cell is black on light gray (229). The cursor is white on dark gray (70). A selection is white on mid gray (120). The active column, the active row, prompts, and a save flash are light gray. Modified and `#` errors are bold white.
- List screens use bold bright white for the active row and reverse video for an error. Muted text stays bright black.

## Risks

- A max-intensity file on `l` is white, the same as a directory. The label still shows size and line count.
- A short selected file on `o` no longer shows the dim floor. It shows the neutral gray instead.
