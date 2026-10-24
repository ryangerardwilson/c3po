# tui Specification

## Purpose
Keeps shortcut text off every interactive c3po screen until the operator presses question mark.
## Requirements
### Requirement: Shortcuts appear only while question mark is open
An interactive screen SHALL NOT paint shortcut text on its listing. Shortcut text SHALL appear only after the operator presses `?`, and `?` SHALL toggle that list. While the list is open, a key that would move, open, copy, delete, add, or remove SHALL NOT do so, and Esc SHALL close the list. On the navigator, deport, Drive browse, the Drive preset list, pass, the tunnel route list, and the looptab job list, `q` SHALL still leave the screen. On the spreadsheet, `q` while the list is open SHALL close the list, and the sheet SHALL leave only through `:q!`. On cyclotree, `q` SHALL leave the screen.

The navigator, deport, Drive browse, the Drive preset list, pass, the tunnel route list, the looptab job list, the spreadsheet, and cyclotree SHALL follow this. `c3po tui l` SHALL NOT paint a shortcut line. The cyclotree call chain and the focused path SHALL stay at the bottom of that screen. The spreadsheet statistics line SHALL stay. A command prompt, a filter prompt, a route name field, a route port field, a route form's save line, and a removal confirmation SHALL stay.

#### Scenario: The navigator hides the legend
- **WHEN** the navigator is listing a directory and the operator has not pressed `?`
- **THEN** the view SHALL NOT contain `j/k move`

#### Scenario: Question mark lists the navigator shortcuts
- **WHEN** the operator presses `?` on the navigator and then presses `?` again
- **THEN** the first view SHALL name cut, copy, paste, and quit
- **AND** the second view SHALL show the directory listing

#### Scenario: Deport hides the legend until question mark
- **WHEN** the operator presses `?` on a deport list, then `j`, then Esc
- **THEN** the view before `?` SHALL NOT contain `j/k move`
- **AND** the open list SHALL name `dd` and quit
- **AND** `j` SHALL leave the cursor on the same port
- **AND** Esc SHALL show the port list again

#### Scenario: Drive browse hides the legend until question mark
- **WHEN** the operator presses `?` on a Drive folder, then `j`, then Esc
- **THEN** the view before `?` SHALL NOT contain `j/k move`
- **AND** the open list SHALL name upload and delete
- **AND** `j` SHALL leave the cursor on the same row
- **AND** Esc SHALL show the folder again

#### Scenario: The preset list hides the legend until question mark
- **WHEN** the operator presses `?` on the Drive preset list, then `j`, then Esc
- **THEN** the view before `?` SHALL NOT contain `j/k move`
- **AND** the open list SHALL name select and quit
- **AND** `j` SHALL leave the cursor on the same preset
- **AND** Esc SHALL show the preset list again

#### Scenario: Pass hides the legend until question mark
- **WHEN** the operator presses `?` on the pass vault list, then `j`, then Esc
- **THEN** the view before `?` SHALL NOT contain `j/k move`
- **AND** the open list SHALL name open and copy
- **AND** the open list SHALL name `dd`
- **AND** `j` SHALL leave the cursor on the same row
- **AND** Esc SHALL show the vault list again

#### Scenario: The tunnel route list hides the legend until question mark
- **WHEN** the operator presses `?` on the tunnel route list, then `j`, then Esc
- **THEN** the view before `?` SHALL NOT contain `a add`
- **AND** the open list SHALL name add, remove, and quit
- **AND** `j` SHALL leave the cursor on the same route
- **AND** Esc SHALL show the route list again
- **AND** the add form SHALL still say `enter save` and `esc cancel`
- **AND** a removal SHALL still ask `y or n`

#### Scenario: The looptab list hides the legend until question mark
- **WHEN** the operator presses `?` on the looptab job list, then `j`, then Esc
- **THEN** the view before `?` SHALL NOT contain `j/k move`
- **AND** the open list SHALL name enter and quit
- **AND** `j` SHALL leave the cursor on the same job
- **AND** Esc SHALL show the job list again

#### Scenario: The spreadsheet keeps shortcuts behind question mark
- **WHEN** the spreadsheet is showing the grid and the operator presses `?`
- **THEN** the grid SHALL NOT show the shortcut list
- **AND** the shortcut list SHALL be shown after `?`

#### Scenario: Cyclotree keeps shortcuts behind question mark
- **WHEN** cyclotree is showing the tree and the operator has not pressed `?`
- **THEN** the tree SHALL NOT show the shortcut list
- **AND** the bottom SHALL show the call chain and the focused path

### Requirement: dd deletes the selected pass item
On `c3po tui pass`, the first `d` SHALL arm deletion and SHALL show `d`. A second `d` SHALL delete the selected item from its open vault and SHALL reload that vault. The cursor SHALL stay at the same index, clamped to the remaining rows. The status SHALL be `deleted` followed by the item title. The secret SHALL stay unread and unprinted. Esc SHALL cancel the arm and SHALL leave the screen open. Any other key SHALL clear the arm and SHALL run as that key. `?` SHALL clear the arm and SHALL open the shortcut list. While the shortcut list is open, `d` SHALL leave the item in place. A vault row SHALL stay, and the status SHALL be `dd deletes an item`. An empty item list SHALL report `no items`. An empty vault list SHALL report `no vaults`. A failed delete SHALL leave the item in place and SHALL show the failure on the status line.

#### Scenario: dd deletes the selected item
- **WHEN** the operator presses `dd` on an item in an open vault
- **THEN** that item SHALL be removed from the vault
- **AND** the listing SHALL show the remaining items
- **AND** the cursor SHALL stay at the same index
- **AND** the status SHALL name the deleted title
- **AND** the view SHALL NOT contain the secret

#### Scenario: Another key cancels the arm
- **WHEN** the operator presses `d` and then `j` on an item list
- **THEN** the item SHALL remain
- **AND** the cursor SHALL move

#### Scenario: Esc cancels the arm
- **WHEN** the operator presses `d` and then Esc on an item
- **THEN** the item SHALL remain
- **AND** the screen SHALL stay open

#### Scenario: dd leaves a vault
- **WHEN** the operator presses `dd` on a vault row
- **THEN** the vault SHALL remain
- **AND** the status SHALL be `dd deletes an item`

#### Scenario: An empty item list reports no items
- **WHEN** the operator presses `dd` in a vault that has no items
- **THEN** the status SHALL be `no items`

#### Scenario: An empty vault list reports no vaults
- **WHEN** the operator presses `dd` and no vaults are listed
- **THEN** the status SHALL be `no vaults`

#### Scenario: A failed delete leaves the item
- **WHEN** deleting the selected item fails
- **THEN** the item SHALL remain
- **AND** the status SHALL report the failure
- **AND** the view SHALL NOT contain the secret

#### Scenario: Help ignores d
- **WHEN** the shortcut list is open and the operator presses `d`
- **THEN** the item SHALL remain
- **AND** the shortcut list SHALL stay open

