# Grayscale screens

## Why

r2d2 screens paint cyan, yellow, red, and green. The terminal program should use black, white, and gray, and should keep a brighter gray where a value or a state used to use a brighter hue.

## What Changes

- `r2d2 l` paints directories white and paints text files and binaries gray. Brightness still scales with line count and size.
- The cyclotree screen paints white text. The plain tree sets no text color.
- `r2d2 o` paints text files gray by line count. The selected row is light gray on dark gray. Muted text and the status line stay gray.
- `r2d2 xl` paints the sheet in white and gray. The focused cell is black on light gray. The cursor is white on dark gray. A selection is white on a mid gray. Modified and error values are bold white.
- The Drive, pass, deport, looptab, and slack lists mark the active row in bold white and an error in reverse video.
- `c3po gmail` and `c3po gcal` are not screens and keep their CLI colors.

## Impact

- Specs: `openspec/specs/tui/spec.md`, `openspec/specs/cyclotree/spec.md`
- Code: `l/`, `cyclotree/`, `o/`, `xl/`, `gdrive/`, `pass/`, `deport/`, `looptab/`, `slack/`
- r2d2 picks this up when it requires the new c3po module.
