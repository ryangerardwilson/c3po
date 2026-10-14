## ADDED Requirements

### Requirement: Prose policy
The global config SHALL accept a top-level `prose` list of globs (default
`**/*.md`, `**/*.markdown`, `**/*.mdx`, `**/*.rst`, `**/*.txt`) and a boolean
`enforcement.count_prose` (default false). A config file that omits them
SHALL use the defaults.

#### Scenario: Omitted prose keys use defaults
- **GIVEN** a config file written before these keys existed
- **WHEN** rgw-ast loads it
- **THEN** `prose` SHALL contain `**/*.md`
- **AND** `enforcement.count_prose` SHALL be false
