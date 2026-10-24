# xl

A terminal grid for a budget. Open an xlsx or a blank sheet, type expenses and income, and keep a formula for savings. Selecting a run of numbers shows the sum, average, median, and mode.

The xlsx stores formulas as formulas, so xl and Excel both keep them. An old csv still opens, and the next save writes xlsx. A workbook can hold more than one sheet, and a formula can use another sheet.

## Keys

Normal mode moves. Insert mode types.

```text
hjkl or arrows    move
i or =            edit the cell
esc               keep the cell and return to normal
enter             edit; down when editing
ctrl-a/e/w        start, end, delete word
alt-f/b           next word, previous word
v then hjkl       select a rectangle
x                 clear the selection
o                 insert a row below
O                 insert a row above
[ ]               previous and next sheet
n                 new sheet
R                 rename the sheet
:                 command line
:w                save
:w path           save to a path
:wq               save and quit
:q!               discard and quit
/                 search this sheet
ctrl-n / ctrl-p   next and previous match
?                 keys and formulas
```

`h`, `j`, `k`, and `l` move only in normal mode. In insert mode they are letters. `n` and `R` are letters while editing. `[` and `]` switch sheets only in normal mode. Enter in normal mode edits the current cell. Enter while editing commits the cell and moves to the cell below in normal mode. While editing, ctrl-a and ctrl-e move to the ends of the cell, alt-f and alt-b move by word, and ctrl-w deletes the word behind the cursor. `:` opens a command line. `:w` saves, and a sheet with no path asks for one. `:wq` saves and quits. `:q!` quits and discards unsaved changes. `/` searches this sheet for text in a cell or in the value it shows, ignoring case. ctrl-n and ctrl-p move to the next and previous match. `o` inserts a blank row below the cursor and `O` inserts one above. The cursor stays on the new row in normal mode. References to this sheet move with the rows. A range grows when the new row is inside it or directly under it.

## Formulas

```text
=E4-B5
=SUM(B2:B4)
=AVERAGE(E2:E3)
=MEDIAN(B2:B4)
=MODE(B2:B4)
=Income!B1
=SUM(Income!B1:B2)
```

Also `MIN`, `MAX`, and `COUNT`. `AVG` is `AVERAGE`, and an xlsx saves it as `AVERAGE`. Ranges can be rows or columns. `+` `-` `*` `/` and parentheses work. A sheet name with a space is quoted, as in `='Food costs'!A1`.

While you type, the characters show in the cell. The focused cell is a cyan shade with black text, and the cursor is red.

When more than one cell is selected, and every filled cell is a number, the bottom line shows `Sum`, `Average`, `Median`, `Mode`, and `n`. A label or other text hides that line. Blank cells are ignored. `Mode none` means no number repeats.

## A budget

```text
A1 Expenses                 D1 Income
A2 Rent        B2 1200      D2 Pay       E2 4000
A3 Food        B3 450       D3 Other     E3 150
A4 Transit     B4 120
A5 Total       B5 =SUM(B2:B4)
                           D4 Total     E4 =SUM(E2:E3)
A7 Savings     B7 =E4-B5
```

`v` then `j` down `B2:B4` shows the expense stats. `B7` shows savings.

The same lists can live on their own sheets. `n` adds one. On the expenses sheet, `=Income!E4-B5` is savings when the income total is `Income!E4`.

## Run

```sh
c3po tui xl
c3po tui xl budget.xlsx
c3po tui xl open budget.xlsx
c3po tui xl help
c3po tui xl version
```

`c3po tui xl` in a terminal opens a blank sheet. Without a terminal it prints help. `c3po tui xl budget` and `c3po tui xl budget.xlsx` create or open `budget.xlsx`. A name that already ends in `.xlsx` is left as it is. `c3po tui xl budget.csv` still loads that csv, and the workbook file is `budget.xlsx`.

## Install

The screen ships with c3po. `install.sh` from this checkout installs `c3po`. There is no `xl` command.

```sh
./install.sh
c3po tui xl help
```
