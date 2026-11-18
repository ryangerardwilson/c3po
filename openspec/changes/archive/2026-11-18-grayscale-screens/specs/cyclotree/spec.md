## MODIFIED Requirements

### Requirement: The tree is calls weighted by cyclomatic complexity
The root SHALL be the named function. Its direct resolved calls SHALL start open. Each scored row SHALL show the function name, its cyclomatic complexity, how many levels of calls can be opened under it, and `path:line`. The root path SHALL be from the directory being read. A nested row SHALL show its path relative to the caller's directory. A relative path that steps upward and is longer than the root path SHALL stay the shorter path. The bottom of the screen SHALL show the call chain from the root down to the row under the cursor, written as `Name -> Name`, and the full path of that row from the directory being read. The plain tree SHALL NOT show that chain or that path. On the screen, the text SHALL be white. The plain tree SHALL NOT set a text color. A level is one call that can be opened. A call back to a function already on the path SHALL NOT add a level. A row with no such call SHALL show depth 0. A call into a directory named `internal` SHALL be listed. Go, Python, TypeScript, and JavaScript calls SHALL be followed. A Rust function SHALL be accepted as a root and SHALL show its cyclomatic complexity. Its calls SHALL NOT be listed. When stdout is not a terminal, the open tree SHALL be printed and the command SHALL exit 0.

#### Scenario: A call into internal is listed
- **GIVEN** `Start` calls `Work`, and `Work` is defined under `internal/`
- **WHEN** the operator runs `r2d2 cyclotree Start` with stdout not a terminal
- **THEN** the text SHALL show `Start` and `Work`
- **AND** the `Work` row SHALL contain `internal/`
- **AND** each row SHALL show that function's cyclomatic complexity

#### Scenario: Depth counts calls that can be opened
- **GIVEN** `Start` calls `Work` and `Work` calls `Start`
- **WHEN** the operator runs `r2d2 cyclotree Start` with stdout not a terminal
- **THEN** the `Start` row SHALL show depth 1
- **AND** the `Work` row SHALL show depth 0

#### Scenario: A Rust root shows its complexity
- **GIVEN** a Rust function `parse_line` with one branch
- **WHEN** the operator runs `r2d2 cyclotree parse_line` with stdout not a terminal
- **THEN** the text SHALL show `parse_line` and its cyclomatic complexity
- **AND** the text SHALL contain one row

#### Scenario: The screen text is white
- **WHEN** the cyclotree screen paints a frame
- **THEN** the text color SHALL be white
- **AND** the plain tree SHALL NOT set a text color
