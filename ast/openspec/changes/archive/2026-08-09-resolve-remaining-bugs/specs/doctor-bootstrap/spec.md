## ADDED Requirements

### Requirement: Doctor and agents-block
`rgw-ast doctor` SHALL report version, config path, root measure, enforcement,
and readiness flags. `rgw-ast agents-block` SHALL print the canonical managed
AGENTS boundary without writing files.

#### Scenario: Agents block is pure output
- **WHEN** agents-block runs
- **THEN** stdout SHALL contain `<!-- rgw-ast:begin -->`
- **AND** no project-local config file SHALL be created
