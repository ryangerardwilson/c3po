# gates Specification

## Purpose
TBD - created by archiving change add-gates. Update Purpose after archive.
## Requirements
### Requirement: Gates publish statutes of provisions
`c3po gates` SHALL print hardcoded legislation so an agent can judge work against it. c3po SHALL NOT scan a repository, call a model, or decide pass or fail. A type is a broad subject. The types SHALL be exactly `code` and then `essays`, in that order. A type SHALL hold zero or more statutes. `code` SHALL hold exactly the statute `architecture`. `essays` SHALL hold no statute. A statute SHALL hold provisions. A provision SHALL have a section, a heading of at most 40 characters, and a body of at most 300 Unicode code points. The legislation SHALL be compiled into the binary. The command SHALL NOT read a config file or a path under the user's home to find the text.

A section SHALL be `S` and then one or more positive integers separated by dots. `S.1.0`, `S.01`, and a bare `1` SHALL NOT be sections. Provisions SHALL be ordered numerically, component by component, and a section that is a prefix of another SHALL come first. A provision SHALL be named by its type, its statute, and its section, for example `code architecture S.1`. There SHALL be no combined id such as `C/A/S.1`. Sections SHALL be case-sensitive.

The `architecture` provisions SHALL be cut from the architecture guardrails text without adding or paraphrasing a rule, with markdown stripped, one bullet or one short paragraph per body. `S.1`, `S.2`, and `S.3` of `architecture` SHALL be `Data model`, `State ownership`, and `Boundaries`, with the bodies `Everything else inherits its shape.`, `Each fact has exactly one owner and one write path.`, and `The core domain is independent of UI, database, and frameworks.` Their subsections SHALL be the goal paragraph, then each hard rule, then each default, then each red flag. `S.4` through `S.8` SHALL be how to apply, severity, checklist, reporting, and exceptions. The hard-rules severity paragraph SHALL be two provisions under `S.5`. Subsections SHALL be numbered from 1 without gaps.

#### Scenario: Sections sort by number
- **GIVEN** provisions with sections `S.10`, `S.3.5.1`, `S.2.3`, `S.3`, `S.1`, and `S.2`
- **WHEN** they are put in section order
- **THEN** the order SHALL be `S.1`, `S.2`, `S.2.3`, `S.3`, `S.3.5.1`, `S.10`

#### Scenario: Every body fits
- **WHEN** every published body is counted with `utf8.RuneCountInString`
- **THEN** no body SHALL be over 300
- **AND** some body SHALL contain `stable, immutable identifier`, `exactly one write path`, `Dependencies MUST point inward`, and `inherits its shape`

### Requirement: Gates refuse unknown names and cannot be rewritten
`list`, `get`, `show`, `edit`, `add`, and `remove` SHALL NOT be commands. `c3po gates list`, `get`, `show`, `edit`, `add`, and `remove`, with or without more words, SHALL exit 2, and stderr SHALL say `unknown command` and that the gates change only when c3po itself is changed and reinstalled. Any other first word that is not a type, help, or version SHALL exit 2 and stderr SHALL say `unknown type`. An unknown statute SHALL exit 2 and stderr SHALL say `unknown statute`. A section that is not published in that statute SHALL exit 2 and stderr SHALL say `unknown section`; `S.9`, `S.1.0`, `S.01`, `1`, `s.1`, and `C/A/S.1` SHALL be unknown sections of `code architecture`. A fourth argument SHALL exit 2. A refusal SHALL print nothing on stdout.

#### Scenario: Unknown names
- **WHEN** the agent runs `c3po gates poems`, `c3po gates code style`, or `c3po gates code architecture S.9`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL contain `unknown type`, `unknown statute`, or `unknown section` in turn
- **AND** stdout SHALL be empty

#### Scenario: The old id is not a section
- **WHEN** the agent runs `c3po gates code architecture C/A/S.1`, `S.1.0`, `S.01`, or `1`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL contain `unknown section`

#### Scenario: A fourth argument
- **WHEN** the agent runs `c3po gates code architecture S.1 S.2`
- **THEN** the process SHALL exit 2
- **AND** stdout SHALL be empty

#### Scenario: List and get are gone
- **WHEN** the agent runs `c3po gates list code` or `c3po gates get C/A/S.1`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL say `unknown command`
- **AND** stdout SHALL be empty

#### Scenario: No show or edit command
- **WHEN** the agent runs `c3po gates show code` or `c3po gates edit S.1`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL say the text changes only when c3po is changed and reinstalled
- **AND** stdout SHALL be empty

### Requirement: Gates probe one level per argument
Each added argument SHALL deepen the same probe. `c3po gates` SHALL print `code` and `essays`, one per line, and SHALL exit 0. `c3po gates <type>` SHALL print the statute names of that type, one per line, and SHALL exit 0; a type with no statute SHALL print `no statutes`. `c3po gates <type> <statute>` SHALL print every provision of that statute in section order, one line each, as the section, two spaces, and the heading, with no banner and no blank line, and SHALL exit 0. The section SHALL be the bare section such as `S.1` or `S.1.1`, so an agent can copy it as the next argument. `c3po gates <type> <statute> <section>` SHALL print only that provision's body and one trailing newline, and SHALL exit 0. It SHALL NOT print the section, the heading, or the sequential rule. No probe output SHALL contain the sequential rule.

#### Scenario: Types in order
- **WHEN** the agent runs `c3po gates`
- **THEN** stdout SHALL be `code` then `essays`
- **AND** the process SHALL exit 0

#### Scenario: The code statutes
- **WHEN** the agent runs `c3po gates code`
- **THEN** stdout SHALL be `architecture`

#### Scenario: Essays have no statutes
- **WHEN** the agent runs `c3po gates essays`
- **THEN** stdout SHALL be `no statutes`
- **AND** the process SHALL exit 0

#### Scenario: The provisions
- **WHEN** the agent runs `c3po gates code architecture`
- **THEN** stdout SHALL start with `S.1  Data model` then `S.1.1  Domain truth`
- **AND** stdout SHALL have one line per provision and no blank line

#### Scenario: One provision
- **WHEN** the agent runs `c3po gates code architecture S.1`
- **THEN** stdout SHALL be `Everything else inherits its shape.` and a newline
- **AND** the process SHALL exit 0

### Requirement: Gates help names the four probes
`c3po gates help`, `c3po gates --help`, and `c3po gates -h` SHALL print help and SHALL exit 0. `c3po help -f gates` SHALL print the same help. The help SHALL show the four probes `c3po gates`, `c3po gates <type>`, `c3po gates <type> <statute>`, and `c3po gates <type> <statute> <section>`. It SHALL name `code` and `essays`, the statute `architecture`, and the 300-character limit. It SHALL NOT name `list`, `get`, or a `C/A/` id. The help SHALL say that architecture is stricter than style, framework choice, or micro-optimization. The help SHALL give the sequential rule: gates run in the list order, a failed gate fails every later gate, and those later gates are not judged. The help SHALL give the report line `S.1  pass` or `S.1  fail`, and after a fail `S.1.1  fail  not judged`. The help SHALL say that the agent judges and c3po only prints the text, that there is no way to edit a gate, and that there is no gates screen. `c3po gates version` SHALL print `c3po <version>` and SHALL exit 0.

#### Scenario: Help
- **WHEN** the agent runs `c3po gates help`
- **THEN** the process SHALL exit 0
- **AND** the text SHALL show `c3po gates <type> <statute> <section>`
- **AND** the text SHALL name `architecture`, `300`, `S.1  pass`, and `not judged`

#### Scenario: Version
- **WHEN** the agent runs `c3po gates version`
- **THEN** stdout SHALL start with `c3po `

