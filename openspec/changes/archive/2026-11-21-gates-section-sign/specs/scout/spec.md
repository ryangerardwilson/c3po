## MODIFIED Requirements

### Requirement: Gates is an app
`c3po gates` SHALL run gates with the remaining arguments. `c3po` and `c3po help` SHALL list `gates` between `deport` and `gcal`. `c3po help -f gates` SHALL print the same text as `c3po gates help`. There SHALL be no gates screen. `c3po tui gates` SHALL exit 2. The usage feature SHALL be `gates`; help and version SHALL stay `help` and `version`. The usage record SHALL NOT store the type, the statute, or the section.

#### Scenario: Gates is listed with the apps
- **WHEN** the operator runs `c3po`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `gates`

#### Scenario: Gates help is the category help
- **WHEN** the operator runs `c3po help -f gates`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL match `c3po gates help`

#### Scenario: Gates usage feature
- **WHEN** the agent runs `c3po gates code architecture §1`
- **THEN** the recorded feature SHALL be `gates`
- **AND** `c3po gates --help` SHALL be recorded as `help`
