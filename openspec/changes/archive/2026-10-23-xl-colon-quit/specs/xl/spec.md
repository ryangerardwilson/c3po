## ADDED Requirements

### Requirement: Commands save and quit
`:w` SHALL save the workbook. `:wq` SHALL save the workbook and SHALL quit. `:q!` SHALL quit and SHALL discard unsaved changes. `q`, ctrl-c, and `:q` SHALL NOT quit. The screen SHALL NOT ask whether to save or discard.

#### Scenario: :wq saves and quits
- **WHEN** the operator runs `:wq` on a modified workbook that has a path
- **THEN** the file SHALL contain the modification
- **AND** the screen SHALL quit

#### Scenario: :q! discards
- **WHEN** the operator runs `:q!` on a modified workbook
- **THEN** the screen SHALL quit
- **AND** the file SHALL NOT gain that modification

#### Scenario: q stays
- **WHEN** the operator presses `q` on a modified workbook
- **THEN** the screen SHALL stay open
- **AND** the modification SHALL remain
