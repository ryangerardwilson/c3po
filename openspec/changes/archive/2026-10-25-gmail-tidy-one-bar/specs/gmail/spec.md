## REMOVED Requirements

### Requirement: Tidy shows one progress line per preset
**Reason**: Tidy keeps one bar for the whole run. A finished line per preset is replaced by that requirement.
**Migration**: Use the one-bar requirement below.

## ADDED Requirements

### Requirement: Tidy shows one progress bar
`c3po gmail tidy` SHALL draw one block progress bar for the whole run. The bar SHALL advance across every preset's sender groups. On a terminal, that bar SHALL update in place. While a preset is in progress, the end of the line SHALL name that preset and email. When the run finishes, the line SHALL remain and SHALL end with one status per preset. Each status SHALL name the preset, the email, and how many messages were trashed. The command SHALL NOT print one line per sender group and SHALL NOT print one bar per preset. The completion line SHALL remain `gmail tidy complete: presets=%d trashed=%d`. With no sender groups, tidy SHALL print no progress bar.

#### Scenario: A terminal redraws one line
- **WHEN** tidy cleans more than one preset on a terminal
- **THEN** the progress SHALL be redrawn on one line
- **AND** the finished line SHALL contain one block bar and each preset's `trashed` count

#### Scenario: A pipe does not print every group
- **WHEN** tidy cleans presets and stdout is not a terminal
- **THEN** the run SHALL produce one progress line
- **AND** the text SHALL NOT contain `processed group`
- **AND** the text SHALL contain `gmail tidy complete: presets=`
