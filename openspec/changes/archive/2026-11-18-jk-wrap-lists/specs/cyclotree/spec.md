## MODIFIED Requirements

### Requirement: hjkl moves through the call tree
`j` and `k` SHALL move among the visible rows and SHALL stay inside the tree. On the last visible row, `j` SHALL move to the first visible row. On the first visible row, `k` SHALL move to the last visible row. An empty tree SHALL leave the cursor in place. `l` on a closed row SHALL open its calls. `l` on an open row SHALL move to its first call. `h` SHALL move to the caller of the row under the cursor. A call back to a function already on the path SHALL be marked `cycle` and SHALL stay closed. `c` SHALL close the row under the cursor and every open row under it, and the cursor SHALL stay on that row. `q` SHALL leave the screen. On the screen, a row that can be opened SHALL be bold and a row that cannot SHALL be dim. The plain tree SHALL NOT use those marks. The cursor marker SHALL occupy a fixed column. Moving the cursor SHALL NOT change the column of a function name.

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

#### Scenario: j and k wrap the visible rows
- **GIVEN** `Start` calls `Work` and both rows are visible
- **WHEN** the operator presses `k` on `Start`
- **THEN** the cursor SHALL be on `Work`
- **AND** `j` SHALL move it back to `Start`
