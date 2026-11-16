## ADDED Requirements

### Requirement: Idealake is an app
`c3po idealake` SHALL run idealake with the remaining arguments. `c3po` and `c3po help` SHALL list `idealake`. `c3po help -f idealake` SHALL print the same text as `c3po idealake help`. `c3po tui idealake` SHALL exit 2.

#### Scenario: Idealake is listed with the apps
- **WHEN** the operator runs `c3po`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `idealake`

#### Scenario: Idealake help is the category help
- **WHEN** the operator runs `c3po help -f idealake`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL match `c3po idealake help`

#### Scenario: Idealake is not a screen
- **WHEN** the operator runs `c3po tui idealake`
- **THEN** the command SHALL exit 2
