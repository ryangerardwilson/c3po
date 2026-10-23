# xl Product Engineer

## Product Contract

`xl` is a local grid for one budget. It is the small part of a spreadsheet that makes a monthly expenses sheet, an income sheet, and a savings formula.

Core loop:

```text
blank sheet or xlsx -> type values and formulas -> xlsx with those formulas
```

The xlsx is the source of truth. Cells store what was typed. A cell that starts with `=` is a formula and is evaluated for display. Saving writes an xlsx workbook, and a formula is stored as a formula cell. An old csv still opens, and the workbook file is the xlsx. `xl budget` and `xl budget.xlsx` create or open `budget.xlsx`. Every sheet in the workbook is saved.

## Keys

Normal mode is the default.

- `hjkl` and the arrow keys move. They do not type.
- `i` or `=` enters insert mode on the current cell. Typed text shows in the cell. The focused cell is a cyan shade with black text, and the cursor is red.
- `esc` keeps the typed text and returns to normal mode on the same cell.
- `enter` in normal mode edits the current cell. While editing, it commits and moves to the cell below in normal mode.
- While editing, `ctrl-a` and `ctrl-e` move to the ends of the cell. `alt-f` and `alt-b` move by word. `ctrl-w` deletes the word behind the cursor.
- `v` then `hjkl` selects a rectangle. Sum, average, median, mode, and the count show when more than one cell is selected and every filled cell is a number. `Mode none` means nothing repeats.
- `x` clears the selection.
- `o` inserts a blank row below the cursor. `O` inserts one above. The cursor stays on the new row in normal mode. References to this sheet move with the rows. A range grows when the new row is inside it or directly under it.
- `[` and `]` move to the previous and next sheet. The cursor on each sheet is kept, and the mode returns to normal.
- `n` adds a sheet. `R` renames the current sheet. Enter confirms the name and stays on the cell. Esc cancels.
- `:` opens a command line. `:w` saves. A sheet with no path asks for one. `:w path` saves to that file. `:wq` saves and quits. `:q` quits.
- `/` searches this sheet. A match is the typed cell or the value it shows. Case is ignored. `ctrl-n` and `ctrl-p` move to the next and previous match.
- `q` quits. A modified workbook asks before discarding.

## Formulas

`SUM`, `AVERAGE` (`AVG`), `MEDIAN`, `MODE`, `MIN`, `MAX`, `COUNT`, cell references, ranges, `+` `-` `*` `/`, and parentheses. A reference can name a sheet: `=Income!B1`, `=SUM(Income!B1:B2)`, `='Food costs'!A1`. This is the whole language.

## Non-goals

No charts, pivot tables, mouse editing, number formats, or deleting a sheet. No GitHub release path. The screen is `c3po tui xl`. The `xl` command is that same binary.

## Identity

- app: xl
- audience: self
- implementation: tui
- why: a budget grid where formulas survive an xlsx save
- what: a terminal workbook for expenses, income, and savings
