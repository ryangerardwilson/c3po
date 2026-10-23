## REMOVED Requirements

### Requirement: The xl name selects the spreadsheet
**Reason**: The spreadsheet is only the `c3po tui xl` screen. The `xl` command is not installed.
**Migration**: Run `c3po tui xl`.

## ADDED Requirements

### Requirement: The program name xl does not select the spreadsheet
Invoking the binary as `xl` SHALL NOT run the spreadsheet. `xl version` SHALL NOT print `0.1.0`. The installer SHALL NOT create an `xl` link. The installer SHALL remove an `xl` symlink that points at the c3po binary.

#### Scenario: The program name is not the spreadsheet
- **WHEN** the operator runs the binary as `xl version`
- **THEN** the text SHALL NOT be `0.1.0`
- **AND** `xl help` SHALL NOT describe the budget grid
