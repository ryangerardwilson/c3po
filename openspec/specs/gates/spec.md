# gates Specification

## Purpose
TBD - created by archiving change add-gates. Update Purpose after archive.
## Requirements
### Requirement: Gates publish statutes of provisions
`c3po gates` SHALL print hardcoded legislation so an agent can judge work against it. c3po SHALL NOT scan a repository, call a model, or decide pass or fail. A type is a broad subject. The types SHALL be exactly `code` and then `essays`, in that order. A type SHALL hold zero or more statutes. `code` SHALL hold exactly the statute `architecture`. `essays` SHALL hold no statute. A statute SHALL hold provisions. A provision SHALL have a section, a heading of at most 40 characters, and a body of at most 300 Unicode code points. The legislation SHALL be compiled into the binary. The command SHALL NOT read a config file or a path under the user's home to find the text.

A section SHALL be `S` and then one or more positive integers separated by dots. `S.1.0`, `S.01`, and a bare `1` SHALL NOT be sections. Provisions SHALL be ordered numerically, component by component, and a section that is a prefix of another SHALL come first. A provision id SHALL be the uppercase first letter of the type, `/`, the uppercase first letter of the statute, `/`, and the section, for example `C/A/S.1`. Ids SHALL be case-sensitive.

The `architecture` provisions SHALL be cut from the architecture guardrails text without adding or paraphrasing a rule, with markdown stripped, one bullet or one short paragraph per body. `C/A/S.1`, `C/A/S.2`, and `C/A/S.3` SHALL be `Data model`, `State ownership`, and `Boundaries`, with the bodies `Everything else inherits its shape.`, `Each fact has exactly one owner and one write path.`, and `The core domain is independent of UI, database, and frameworks.` Their subsections SHALL be the goal paragraph, then each hard rule, then each default, then each red flag. `S.4` through `S.8` SHALL be how to apply, severity, checklist, reporting, and exceptions. The hard-rules severity paragraph SHALL be two provisions under `S.5`. Subsections SHALL be numbered from 1 without gaps.

#### Scenario: Sections sort by number
- **GIVEN** provisions with sections `S.10`, `S.3.5.1`, `S.2.3`, `S.3`, `S.1`, and `S.2`
- **WHEN** they are put in section order
- **THEN** the order SHALL be `S.1`, `S.2`, `S.2.3`, `S.3`, `S.3.5.1`, `S.10`

#### Scenario: Every body fits
- **WHEN** every published body is counted with `utf8.RuneCountInString`
- **THEN** no body SHALL be over 300
- **AND** some body SHALL contain `stable, immutable identifier`, `exactly one write path`, `Dependencies MUST point inward`, and `inherits its shape`

### Requirement: Gates list and get
`c3po gates` and `c3po gates list` SHALL print `code` and `essays`, one per line, and SHALL exit 0. `c3po gates list <type>` SHALL print the statute names of that type, one per line, and SHALL exit 0; a type with no statute SHALL print `no statutes`. `c3po gates list <type> <statute>` SHALL print every provision of that statute in section order, one line each, as the id, two spaces, and the heading, with no banner and no blank line, and SHALL exit 0. `c3po gates get <id>` SHALL print only that provision's body and one trailing newline, and SHALL exit 0. It SHALL NOT print the id, the heading, or the sequential rule. List and get output SHALL NOT contain the sequential rule.

#### Scenario: Types in order
- **WHEN** the agent runs `c3po gates`
- **THEN** stdout SHALL be `code` then `essays`
- **AND** the process SHALL exit 0

#### Scenario: The code statutes
- **WHEN** the agent runs `c3po gates list code`
- **THEN** stdout SHALL be `architecture`

#### Scenario: Essays have no statutes
- **WHEN** the agent runs `c3po gates list essays`
- **THEN** stdout SHALL be `no statutes`
- **AND** the process SHALL exit 0

#### Scenario: The provisions
- **WHEN** the agent runs `c3po gates list code architecture`
- **THEN** stdout SHALL start with `C/A/S.1  Data model` then `C/A/S.1.1  Domain truth`
- **AND** stdout SHALL have one line per provision and no blank line

#### Scenario: One provision
- **WHEN** the agent runs `c3po gates get C/A/S.1`
- **THEN** stdout SHALL be `Everything else inherits its shape.` and a newline
- **AND** the process SHALL exit 0

### Requirement: Gates refuse unknown names and cannot be rewritten
An unknown type SHALL exit 2 and stderr SHALL say `unknown type`. An unknown statute SHALL exit 2 and stderr SHALL say `unknown statute`. An id that is not published, including `c/a/s.1`, SHALL exit 2 and stderr SHALL say `unknown gate`. `show` SHALL NOT be a command. `c3po gates edit`, `add`, and `remove` SHALL exit 2, and stderr SHALL say that the gates change only when c3po itself is changed and reinstalled. Every other unknown subcommand SHALL exit 2. Extra words after `list <type> <statute>` or `get <id>`, and `get` with no id, SHALL exit 2. A refusal SHALL print nothing on stdout.

#### Scenario: Unknown names
- **WHEN** the agent runs `c3po gates list poems`, `c3po gates list code style`, or `c3po gates get c/a/s.1`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL contain `unknown type`, `unknown statute`, or `unknown gate` in turn
- **AND** stdout SHALL be empty

#### Scenario: No show or edit command
- **WHEN** the agent runs `c3po gates show code` or `c3po gates edit C/A/S.1`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL say the text changes only when c3po is changed and reinstalled

### Requirement: Gates help names list and get
`c3po gates help`, `c3po gates --help`, and `c3po gates -h` SHALL print help and SHALL exit 0. `c3po help -f gates` SHALL print the same help. The help SHALL name `list`, `get`, `code` and `essays`, the statute `architecture`, an id shaped `C/A/S.1`, and the 300-character limit. The help SHALL say that architecture is stricter than style, framework choice, or micro-optimization. The help SHALL give the sequential rule: gates run in the list order, a failed gate fails every later gate, and those later gates are not judged. The help SHALL give the report line `<id>  pass` or `<id>  fail`, and after a fail `<id>  fail  not judged`. The help SHALL say that the agent judges and c3po only prints the text, that there is no way to edit a gate, and that there is no gates screen. `c3po gates version` SHALL print `c3po <version>` and SHALL exit 0.

#### Scenario: Help
- **WHEN** the agent runs `c3po gates help`
- **THEN** the process SHALL exit 0
- **AND** the text SHALL name `list`, `get`, `architecture`, `C/A/S.1`, `300`, and `not judged`

#### Scenario: Version
- **WHEN** the agent runs `c3po gates version`
- **THEN** stdout SHALL start with `c3po `

