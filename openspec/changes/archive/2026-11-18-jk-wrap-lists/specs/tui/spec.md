## ADDED Requirements

### Requirement: j and k wrap a screen list
On the Drive file list, `j` and `k` SHALL wrap. On the Drive preset list, `j` and `k` SHALL wrap. On the pass vault list and the pass item list, `j` and `k` SHALL wrap. On the last row, `j` SHALL move to the first row. On the first row, `k` SHALL move to the last row. A list of one row SHALL keep the cursor on that row. An empty list SHALL leave the cursor in place. `g` and `G` SHALL still move to the first and last row. While the shortcut list is open, `j` and `k` SHALL NOT move. Deleting a pass item or a Drive file SHALL keep the cursor at the same index, clamped to the remaining rows.

#### Scenario: j and k wrap the Drive file list
- **WHEN** the file list has two files and the cursor is on the second
- **AND** the operator presses `j`
- **THEN** the cursor SHALL be on the first file
- **AND** `k` SHALL move it to the second file

#### Scenario: j and k wrap the Drive preset list
- **WHEN** the preset list has two presets and the cursor is on the first
- **AND** the operator presses `k`
- **THEN** the cursor SHALL be on the second preset
- **AND** `j` SHALL move it to the first preset

#### Scenario: j and k wrap the pass list
- **WHEN** the item list has two items and the cursor is on the first
- **AND** the operator presses `k`
- **THEN** the cursor SHALL be on the second item
- **AND** `j` SHALL move it to the first item

#### Scenario: Question mark blocks the wrap
- **WHEN** the shortcut list is open on the pass screen
- **AND** the operator presses `j`
- **THEN** the cursor SHALL stay on the same row

#### Scenario: A delete clamps instead of wrapping
- **WHEN** the operator deletes the selected pass item
- **THEN** the cursor SHALL stay at the same index, clamped to the remaining rows
