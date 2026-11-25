## REMOVED Requirements

### Requirement: Gates publish statutes of provisions
**Reason**: The operator keeps these rules as r2d2 skills. A c3po command that prints them is not needed.
**Migration**: `c3po gates` is an unknown command and exits 2. The rules live in r2d2 skills.

### Requirement: Gates refuse unknown names and cannot be rewritten
**Reason**: The operator keeps these rules as r2d2 skills. A c3po command that prints them is not needed.
**Migration**: `c3po gates` is an unknown command and exits 2. The rules live in r2d2 skills.

### Requirement: Gates probe one level per argument
**Reason**: The operator keeps these rules as r2d2 skills. A c3po command that prints them is not needed.
**Migration**: `c3po gates` is an unknown command and exits 2. The rules live in r2d2 skills.

### Requirement: Gates help names the four probes
**Reason**: The operator keeps these rules as r2d2 skills. A c3po command that prints them is not needed.
**Migration**: `c3po gates` is an unknown command and exits 2. The rules live in r2d2 skills.

### Requirement: Gates tally a sheet
**Reason**: The operator keeps these rules as r2d2 skills. A c3po command that prints them is not needed.
**Migration**: `c3po gates` is an unknown command and exits 2. The rules live in r2d2 skills.

## ADDED Requirements

### Requirement: Gates is not a command
`c3po gates` SHALL be an unknown command and SHALL exit 2. `c3po gates --help` and `c3po gates version` SHALL also be unknown commands and SHALL exit 2. `c3po` and `c3po help` SHALL NOT list `gates`. The usage feature SHALL be `unknown`. There SHALL be no gates screen.

#### Scenario: The name is rejected
- **WHEN** the operator runs `c3po gates`
- **THEN** the command SHALL exit 2
- **AND** the text SHALL say unknown command

#### Scenario: Help and version of that name are rejected
- **WHEN** the operator runs `c3po gates --help`
- **THEN** the command SHALL exit 2
- **AND** `c3po gates version` SHALL exit 2

#### Scenario: The apps list omits gates
- **WHEN** the operator runs `c3po`
- **THEN** the text SHALL NOT list `gates`
