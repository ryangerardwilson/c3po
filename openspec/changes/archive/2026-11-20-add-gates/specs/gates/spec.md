## ADDED Requirements

### Requirement: Gates print hardcoded legislation
`c3po gates` SHALL print hardcoded legislation so an agent can judge work against it. c3po SHALL NOT scan a repository, call a model, or decide pass or fail. A type is a broad subject. The types SHALL be exactly `code` and then `essays`, in that order. A gate SHALL belong to one type and SHALL have a section number, a title, and a body. A section number SHALL be one or more positive integers separated by dots. Gates of a type SHALL be ordered numerically, component by component, and a number that is a prefix of another SHALL come first. The only published gate SHALL be code section `1`, titled `Architecture guardrails`. Its body SHALL be the file `gates/statute/code/1.md`, embedded in the binary with `go:embed`. The command SHALL NOT read a config file or a path under the user's home to find the text.

Whenever a type's gates are listed or shown as a set, the output SHALL include this line on its own: `Gates of a type run in section order. A failed gate fails every later gate, and those later gates are not judged.`

#### Scenario: Sections sort by number
- **GIVEN** gates with sections `10`, `3.5.1`, `2.3`, `3`, `1`, and `2`
- **WHEN** they are put in section order
- **THEN** the order SHALL be `1`, `2`, `2.3`, `3`, `3.5.1`, `10`

#### Scenario: The body is compiled in
- **WHEN** the operator runs `c3po gates show code 1`
- **THEN** the body SHALL equal `gates/statute/code/1.md` byte for byte
- **AND** it SHALL contain `stable, immutable identifier`

### Requirement: Gates list and show
`c3po gates` and `c3po gates list` SHALL print `code` and `essays`, one per line, and SHALL exit 0. `c3po gates list <type>` SHALL print the sequential line and then one line `Section <number>  <title>` per gate in section order, and SHALL exit 0. `c3po gates show <type>` SHALL print the sequential line and then every gate of that type in section order, and SHALL exit 0. `c3po gates show <type> <section>` SHALL print that one gate without the sequential line and SHALL exit 0. A gate SHALL print as `Section <number>`, the title, a blank line, and the body unchanged. A type with no gates SHALL print the sequential line and `no gates` and SHALL exit 0.

#### Scenario: Types in order
- **WHEN** the agent runs `c3po gates`
- **THEN** stdout SHALL be `code` then `essays`
- **AND** the process SHALL exit 0

#### Scenario: The code list
- **WHEN** the agent runs `c3po gates list code`
- **THEN** stdout SHALL be the sequential line then `Section 1  Architecture guardrails`

#### Scenario: One gate
- **WHEN** the agent runs `c3po gates show code 1`
- **THEN** stdout SHALL start with `Section 1`, `Architecture guardrails`, and a blank line
- **AND** stdout SHALL NOT contain the sequential line

#### Scenario: Essays have no gates
- **WHEN** the agent runs `c3po gates list essays` or `c3po gates show essays`
- **THEN** stdout SHALL be the sequential line then `no gates`
- **AND** the process SHALL exit 0

### Requirement: Gates refuse unknown input and cannot be rewritten
An unknown type SHALL exit 2 and stderr SHALL say `unknown type`. A section the type does not publish, including `2` and `1.0` for code, SHALL exit 2 and stderr SHALL say `unknown section`. There SHALL be no config file, no user directory, and no add, edit, or remove command. `c3po gates edit` and every other unknown subcommand SHALL exit 2, and stderr SHALL say that the gates change only when c3po itself is changed and reinstalled. Extra words after `list <type>` or `show <type> <section>`, and `show` with no type, SHALL exit 2. A refusal SHALL print nothing on stdout.

#### Scenario: Unknown type
- **WHEN** the agent runs `c3po gates show poems`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL contain `unknown type`

#### Scenario: Unknown section
- **WHEN** the agent runs `c3po gates show code 2` or `c3po gates show code 1.0`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL contain `unknown section`

#### Scenario: No edit command
- **WHEN** the agent runs `c3po gates edit code 1`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL say the text changes only when c3po is changed and reinstalled

### Requirement: Gates help and version
`c3po gates help`, `c3po gates --help`, and `c3po gates -h` SHALL print help and SHALL exit 0. `c3po help -f gates` SHALL print the same help. The help SHALL name `code` and `essays`, `list`, `show`, the sequential line, and that the text changes only when c3po is changed and reinstalled. The help SHALL say that the agent judges and c3po only prints the text. The help SHALL give the report line `Section <number>  <title>  pass` or `fail`, and SHALL say that after a fail each later section is `fail` and `not judged`. `c3po gates version` SHALL print `c3po <version>` and SHALL exit 0. There SHALL be no gates screen.

#### Scenario: Help
- **WHEN** the agent runs `c3po gates help`
- **THEN** the process SHALL exit 0
- **AND** the text SHALL name `code`, `essays`, `list`, `show`, and `not judged`

#### Scenario: Version
- **WHEN** the agent runs `c3po gates version`
- **THEN** stdout SHALL start with `c3po `
