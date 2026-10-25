## ADDED Requirements

### Requirement: Tidy shows one progress line per preset
While `c3po gmail tidy` cleans a preset, it SHALL draw one progress line for that preset. The line SHALL show the preset, the account email, a block bar, and how many sender groups are done out of the total. On a terminal, that line SHALL update in place. When the preset finishes, the line SHALL remain and SHALL show how many messages were trashed. The command SHALL NOT print one line per sender group. The completion line SHALL remain `gmail tidy complete: presets=%d trashed=%d`.

#### Scenario: A terminal redraws one line
- **WHEN** tidy cleans a preset on a terminal
- **THEN** the progress SHALL be redrawn on the same line
- **AND** the finished line SHALL contain the email, a block, and `trashed`

#### Scenario: A pipe does not print every group
- **WHEN** tidy cleans a preset and stdout is not a terminal
- **THEN** that preset SHALL produce one line
- **AND** the text SHALL NOT contain `processed group`
- **AND** the text SHALL contain `gmail tidy complete: presets=`
