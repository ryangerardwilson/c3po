# cyclotree

## Purpose

cyclotree is a screen of the calls one function makes, or of the whole application from its entrypoint. Each row is a function, its cyclomatic complexity, and how many levels of calls can be opened under it. The operator moves through that tree with hjkl. While the screen is open, the tree updates as source files change.

## Requirements

### Requirement: Cyclotree is an app
`c3po cyclotree [symbol] [dir]` SHALL read `[dir]`, defaulting to the working directory. `[symbol]` MAY come before or after `[dir]` and MAY be omitted. `c3po cyclotree help` SHALL exit 0 and SHALL describe `j`, `k`, `h`, and `l`. An unknown flag or two symbols SHALL exit 2 and SHALL print the help. A path given as the directory that is not a directory SHALL exit 1. An unknown function SHALL exit 1 and SHALL name the symbol. When more than one function has the name, the command SHALL exit 1 and SHALL list each `path:line name`.

#### Scenario: Help names the motions
- **WHEN** the operator runs `c3po cyclotree help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL describe `j`, `k`, `h`, and `l`

#### Scenario: An unknown function is an error
- **WHEN** the operator runs `c3po cyclotree Missing` in an empty directory
- **THEN** the command SHALL exit 1
- **AND** the text SHALL contain `no symbol "Missing"`

### Requirement: No symbol starts at the entrypoint
When no symbol is given, the command SHALL start at the program entrypoint and SHALL include every resolved call reached from it. A Go entrypoint SHALL be `func main` in package `main`. A Python entrypoint SHALL be a module-level function named `main`. A JavaScript or TypeScript entrypoint SHALL be a function named `main`. A Rust entrypoint SHALL be `fn main`, and its calls SHALL NOT be listed. A call back to a function already on the path SHALL be marked `cycle` and SHALL stay closed. A function already opened on another branch SHALL stay closed. When no entrypoint exists, the command SHALL exit 1 and the text SHALL contain `no entrypoint`. When more than one entrypoint exists, the command SHALL exit 1 and SHALL list each `path:line name`.

#### Scenario: The application is treed from main
- **GIVEN** `main` calls `Start` and `Start` calls `Work` in package `main`
- **WHEN** the operator runs `c3po cyclotree` with no symbol and stdout is not a terminal
- **THEN** the command SHALL exit 0
- **AND** the text SHALL show `main`, `Start`, and `Work`
- **AND** `Work` SHALL be nested under `Start`

#### Scenario: A repeated call stays closed
- **GIVEN** `main` calls `A` and `B`, and both `A` and `B` call `C`, and `C` calls `D`
- **WHEN** the operator runs `c3po cyclotree` with no symbol and stdout is not a terminal
- **THEN** `D` SHALL appear once, under the first `C`
- **AND** the other `C` SHALL show no further calls

#### Scenario: No entrypoint is an error
- **WHEN** the operator runs `c3po cyclotree` in an empty directory
- **THEN** the command SHALL exit 1
- **AND** the text SHALL contain `no entrypoint`

#### Scenario: Two entrypoints are listed
- **GIVEN** two functions named `main` in package `main`
- **WHEN** the operator runs `c3po cyclotree` with no symbol
- **THEN** the command SHALL exit 1
- **AND** the text SHALL list both paths

### Requirement: The tree is calls weighted by cyclomatic complexity
The root SHALL be the named function. Its direct resolved calls SHALL start open. Each scored row SHALL show the function name, its cyclomatic complexity, how many levels of calls can be opened under it, and `path:line`. The root path SHALL be from the directory being read. A nested row SHALL show its path relative to the caller's directory. A relative path that steps upward and is longer than the root path SHALL stay the shorter path. The bottom of the screen SHALL show the call chain from the root down to the row under the cursor, written as `Name -> Name`, and the full path of that row from the directory being read. The plain tree SHALL NOT show that chain or that path. On the screen, the text SHALL be yellow. The plain tree SHALL NOT be yellow. A level is one call that can be opened. A call back to a function already on the path SHALL NOT add a level. A row with no such call SHALL show depth 0. A call into a directory named `internal` SHALL be listed. Go, Python, TypeScript, and JavaScript calls SHALL be followed. A Rust function SHALL be accepted as a root and SHALL show its cyclomatic complexity. Its calls SHALL NOT be listed. When stdout is not a terminal, the open tree SHALL be printed and the command SHALL exit 0.

#### Scenario: A call into internal is listed
- **GIVEN** `Start` calls `Work`, and `Work` is defined under `internal/`
- **WHEN** the operator runs `c3po cyclotree Start` with stdout not a terminal
- **THEN** the text SHALL show `Start` and `Work`
- **AND** the `Work` row SHALL contain `internal/`
- **AND** each row SHALL show that function's cyclomatic complexity

#### Scenario: Depth counts calls that can be opened
- **GIVEN** `Start` calls `Work` and `Work` calls `Start`
- **WHEN** the operator runs `c3po cyclotree Start` with stdout not a terminal
- **THEN** the `Start` row SHALL show depth 1
- **AND** the `Work` row SHALL show depth 0

#### Scenario: A Rust root shows its complexity
- **GIVEN** a Rust function `parse_line` with one branch
- **WHEN** the operator runs `c3po cyclotree parse_line` with stdout not a terminal
- **THEN** the text SHALL show `parse_line` and its cyclomatic complexity
- **AND** the text SHALL contain one row

### Requirement: hjkl moves through the call tree
`j` and `k` SHALL move among the visible rows and SHALL stay inside the tree. `l` on a closed row SHALL open its calls. `l` on an open row SHALL move to its first call. `h` SHALL move to the caller of the row under the cursor. A call back to a function already on the path SHALL be marked `cycle` and SHALL stay closed. `c` SHALL close the row under the cursor and every open row under it, and the cursor SHALL stay on that row. `q` SHALL leave the screen. On the screen, a row that can be opened SHALL be bold and a row that cannot SHALL be dim. The plain tree SHALL NOT use those marks. The cursor marker SHALL occupy a fixed column. Moving the cursor SHALL NOT change the column of a function name.

#### Scenario: A cycle stays closed
- **GIVEN** `Start` calls `Work` and `Work` calls `Start`
- **WHEN** the operator moves to `Work` and presses `l`
- **THEN** the opened row SHALL be marked `cycle`
- **AND** pressing `l` on that cycle SHALL leave it closed
- **AND** `h` SHALL move back to `Work`, then to `Start`

#### Scenario: c collapses what is open under the row
- **GIVEN** `Start` calls `Work` and `Work` calls `Start`
- **WHEN** the operator moves to `Work`, presses `l`, then presses `c`
- **THEN** the cycle row SHALL be gone
- **AND** `Work` SHALL remain

#### Scenario: Moving the cursor does not shift names
- **GIVEN** `Start` calls `Work`
- **WHEN** the operator moves from `Start` to `Work`
- **THEN** the column of `Start` and the column of `Work` SHALL be unchanged

#### Scenario: A leaf is dim before it is opened
- **GIVEN** `main` calls `Leaf` and `Leaf` calls nothing
- **WHEN** the screen is shown and the operator has not moved
- **THEN** `Leaf` SHALL be dim

### Requirement: The screen follows source changes
While the screen is open, cyclotree SHALL rebuild the tree when a source file under the directory changes. The rebuild SHALL keep each function open or closed as the operator left it, and SHALL keep the cursor on the same function. A function that is new SHALL use the tree's ordinary open state. When the rebuild fails, the current screen SHALL stay. When stdout is not a terminal, the command SHALL print the tree and SHALL NOT wait for changes.

#### Scenario: A collapsed row stays collapsed after a change
- **GIVEN** the operator collapsed a row
- **WHEN** a source file in the directory changes
- **THEN** that row SHALL stay collapsed
- **AND** the cursor SHALL stay on the same function

### Requirement: Enter opens the file in vim
Enter on a row SHALL open that row's file in vim at the row's line. The working directory SHALL be the directory cyclotree read. When vim exits, the screen SHALL return. Enter on a row with no file SHALL leave the screen as it is.

#### Scenario: Enter opens the row's file
- **GIVEN** a row shows a file and a line
- **WHEN** the operator presses enter
- **THEN** vim SHALL be started on that file at that line
- **AND** the screen SHALL return when vim exits

### Requirement: Question mark toggles the shortcut list
The screen SHALL NOT show the shortcut list until the operator presses `?`. `?` SHALL toggle that list. While the list is open, `j`, `k`, `h`, `l`, `c`, and enter SHALL NOT change the tree. Pressing `?` again SHALL close the list and SHALL show the tree as it was. `q` SHALL still leave the screen.

#### Scenario: The tree hides the shortcut list
- **WHEN** the screen is shown and the operator has not pressed `?`
- **THEN** the screen SHALL NOT show the shortcut list

#### Scenario: Question mark opens and closes the list
- **GIVEN** the cursor is on a row
- **WHEN** the operator presses `?` and then presses `?` again
- **THEN** the shortcut list SHALL be shown after the first press
- **AND** the tree SHALL return after the second press
- **AND** the cursor SHALL be on the same row

#### Scenario: A nested file is relative to its caller
- **GIVEN** `main` in `main.go` calls `Start` in `work/work.go` and `Start` calls `Hidden` in that same file
- **WHEN** the operator runs `c3po cyclotree` with stdout not a terminal
- **THEN** the `Start` row SHALL show `work/work.go`
- **AND** the `Hidden` row SHALL show `work.go`

#### Scenario: The bottom line shows the full path
- **GIVEN** `main` in `main.go` calls `Start` in `work/work.go` and `Start` calls `Hidden` in that same file
- **WHEN** the cursor is on `Hidden`
- **THEN** the row SHALL show `work.go`
- **AND** the bottom SHALL show `main -> Start -> Hidden`
- **AND** the bottom line SHALL show `work/work.go`
