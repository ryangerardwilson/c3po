## MODIFIED Requirements

### Requirement: Tidy shows one progress bar
`c3po gmail tidy` SHALL draw one block progress bar for the whole run. The bar SHALL advance across every preset's sender groups. On a terminal, that bar SHALL update in place. The bar SHALL NOT name a preset or an email. When the run finishes, the bar line SHALL remain and SHALL show the bar and how many sender groups are done. Tidy SHALL then print a table with one row per preset. Each row SHALL name the preset, the email, and how many messages were trashed. The command SHALL NOT print one line per sender group and SHALL NOT print one bar per preset. The completion line SHALL remain `gmail tidy complete: presets=%d trashed=%d`. With no sender groups, tidy SHALL print no progress bar and SHALL print no table.

#### Scenario: A terminal redraws one line
- **WHEN** tidy cleans more than one preset on a terminal
- **THEN** the progress SHALL be redrawn on one line
- **AND** that line SHALL NOT name an email
- **AND** the output SHALL contain a table row for each preset

#### Scenario: A pipe does not print every group
- **WHEN** tidy cleans presets and stdout is not a terminal
- **THEN** the run SHALL produce one progress line
- **AND** the text SHALL NOT contain `processed group`
- **AND** the text SHALL contain a table header for preset, email, and trashed
- **AND** the text SHALL contain `gmail tidy complete: presets=`
