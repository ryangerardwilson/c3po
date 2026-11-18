## ADDED Requirements

### Requirement: The slack screen snapshots its place
While `r2d2 slack` is open it SHALL publish the c3po database about every 10 minutes, and it SHALL publish once when the screen leaves. That publish SHALL wait until the upload finishes or 60 seconds pass. A key that stores the place SHALL NOT publish. The stored place SHALL name the screen `slack` and SHALL NOT contain the user token.

When the runtime database is missing, the screen SHALL load the last published backup before it reads the place. When that load fails for a reason other than the object being absent, the screen SHALL NOT open and SHALL NOT write a session. `--resume <id>` SHALL restore the workspace and row from the loaded database.

#### Scenario: Moving does not publish
- **WHEN** the operator moves the cursor on the slack screen
- **THEN** the place SHALL be stored in the runtime database
- **AND** that key SHALL NOT publish the backup

#### Scenario: Leaving publishes once
- **WHEN** the operator leaves the slack screen
- **THEN** the screen SHALL publish the c3po database once
- **AND** it SHALL wait for that publish

#### Scenario: Resume uses the loaded backup
- **GIVEN** the runtime file was missing and the backup holds a slack place
- **WHEN** the operator runs `r2d2 slack --resume <id>`
- **THEN** the screen SHALL open that workspace and row
- **AND** the place SHALL NOT contain the user token
