## Context

`o` inserts a blank row below the cursor and `O` inserts one above. Later rows shift down, and references to this sheet move with them. A range grows when the new row is inside it or directly under it. `x` clears the selected cells. Drive, deport, and the navigator use `dd` as an armed chord.

## Goals / Non-Goals

- Delete the current row with `dd` in normal mode.
- Arm on the first `d`, show `d`, cancel with Esc, and let any other key run normally.
- Shift later rows up and rewrite formulas on every sheet that names this one.
- Leave `o`, `O`, and `x` unchanged.
- Leave `d` typed into a cell as text.

## Decisions

- The chord matches the other screens. The title shows `d` while the chord is armed. The second `d` deletes only in normal mode. Esc clears the arm. Any other key, including `j` and `q`, clears the arm and runs as itself.
- `DeleteRow` removes that row's cells, moves later rows up one, and decrements the row count while it is above one.
- A formula reference to the deleted row is rewritten to `#REF!`, which displays as `#REF`. A range that contained the row shrinks by one. A range entirely below the row shifts up. A range entirely above stays. Text that is not a formula is left as text and moves with its row.
- The same rewrite applies to formulas on other sheets that name this sheet.

## Risks / Trade-offs

- A row delete is permanent after the second `d`. The arm is the confirmation.
- The help list gains one line.
