## ADDED Requirements

### Requirement: Feature usage is recorded
Every c3po invocation SHALL add one to a daily counter for one catalog feature and one actor. The actor SHALL be `human`, `agent`, or `service`. `C3PO_ACTOR` SHALL select the actor when its value is exactly one of those three words. The counter SHALL NOT store arguments, paths, presets, ports, or message text. Help and version SHALL be recorded as `help` and `version`, including `c3po help -f ast patch` and `c3po ast patch --help`. A failure to write the counter SHALL NOT change the command's exit code and SHALL NOT print on stderr. `c3po help` SHALL name `c3po usage`. `c3po usage` SHALL exit 0 and SHALL list every catalog feature that has no recorded use, then the features that have been used, with separate human, agent, and service counts, the number of exits 0, and the UTC date of the last use. A feature that was used and never exited 0 SHALL appear under used. The report SHALL be written before that same invocation is added to the counter. A stored feature that is not in the catalog SHALL be listed after the used features.

#### Scenario: An unused feature is listed
- **WHEN** the operator runs `c3po usage` and `ast cut` has never been recorded
- **THEN** the command SHALL exit 0
- **AND** the text SHALL list `ast cut` under `unused`

#### Scenario: Human and agent counts stay separate
- **GIVEN** one successful run recorded as a human and one successful run of the same feature recorded as an agent
- **WHEN** the operator runs `c3po usage`
- **THEN** that feature's row SHALL show both counts

#### Scenario: Help does not count as the feature
- **WHEN** the operator runs `c3po help -f ast patch`
- **THEN** the recorded feature SHALL be `help`
- **AND** `ast patch` SHALL remain unused until some invocation records `ast patch`

#### Scenario: A recording failure leaves the exit code alone
- **WHEN** the usage record cannot be written during `c3po version`
- **THEN** the command SHALL exit 0
- **AND** stderr SHALL NOT mention the usage record
