## MODIFIED Requirements

### Requirement: Gates probe one level per argument
Each added argument after the type and statute SHALL deepen the same probe. `c3po gates` SHALL print `code` and `essays`, one per line, and SHALL exit 0. `c3po gates <type>` SHALL print the statute names of that type, one per line, and SHALL exit 0; a type with no statute SHALL print `no statutes`. `c3po gates <type> <statute>` SHALL print every provision of that statute in section order, one line each, as the section, two spaces, and the heading, with no banner and no blank line, and SHALL exit 0. The section SHALL be the published section such as `§1` or `§1.1`, so an agent can copy it as the next argument.

An immediate subsection of a section SHALL be a published section of the same statute that is that section, a dot, and one more positive integer. `§8.1` SHALL be an immediate subsection of `§8`. `§8.1.1` SHALL be an immediate subsection of `§8.1` and SHALL NOT be an immediate subsection of `§8`. A section with no immediate subsection is a leaf.

`c3po gates <type> <statute> <section>` SHALL exit 0 whichever form the section is written in. For a leaf it SHALL print only that provision's body and one trailing newline. For a section with immediate subsections it SHALL print the body, a blank line, and then one line per immediate subsection in section order, as the subsection's published section, two spaces, and its heading, with no banner and no trailing blank line. It SHALL NOT print a subsection's body or any deeper section. It SHALL NOT print the requested section, its heading, or the sequential rule.

`c3po gates <type> <statute> <section> <section>...` SHALL print every requested provision in one pass, in the order written and not in statute order, and SHALL exit 0. Each block SHALL be the published section on its own line, then the body on the next line, then, when the section has immediate subsections, a blank line and one subsection line each in the same shape. A leaf block SHALL have no subsection lines. A blank line SHALL separate one block from the next. The last line SHALL be followed by one newline and no blank line. No heading and no probe output SHALL contain the sequential rule.

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
- **THEN** stdout SHALL start with `§1  Data model` then `§1.1  Domain truth`
- **AND** stdout SHALL have one line per provision and no blank line

#### Scenario: One provision
- **WHEN** the agent runs `c3po gates code architecture §8.3`, `8.3`, or `S.8.3`
- **THEN** stdout SHALL be `Mention it in the summary of the change.` and a newline
- **AND** the process SHALL exit 0

#### Scenario: A provision with subsections
- **WHEN** the agent runs `c3po gates code architecture 8`, `§8`, or `S.8`
- **THEN** stdout SHALL be `A rule may be broken deliberately, never silently.`, a blank line, `§8.1  Comment at the site`, `§8.2  Contain the violation`, `§8.3  Mention it in the summary`, and `§8.4  Legitimate exceptions`, each followed by a newline
- **AND** stdout SHALL NOT contain the body of `§8.1`
- **AND** the process SHALL exit 0

#### Scenario: Subsections list headings, not bodies
- **WHEN** the agent runs `c3po gates code architecture 1`
- **THEN** stdout SHALL contain the line `§1.1  Domain truth`
- **AND** stdout SHALL NOT contain the body of `§1.1`

#### Scenario: Only the next level is listed
- **GIVEN** a statute with `§8`, `§8.1`, `§8.1.1` through `§8.1.6`, `§8.1.2.3`, and `§8.2`
- **WHEN** `§8` is printed
- **THEN** the subsection lines SHALL be `§8.1` and `§8.2` only
- **AND** stdout SHALL NOT contain `§8.1.1` or `§8.1.2.3`
- **WHEN** `§8.1` is printed
- **THEN** the subsection lines SHALL be `§8.1.1` through `§8.1.6` only

#### Scenario: Several provisions in the order written
- **WHEN** the agent runs `c3po gates code architecture 8.3 8`
- **THEN** stdout SHALL be `§8.3`, `Mention it in the summary of the change.`, a blank line, `§8`, `A rule may be broken deliberately, never silently.`, a blank line, `§8.1  Comment at the site`, `§8.2  Contain the violation`, `§8.3  Mention it in the summary`, and `§8.4  Legitimate exceptions`, each followed by a newline
- **AND** stdout SHALL NOT end with a blank line
- **AND** the process SHALL exit 0

#### Scenario: Several provisions keep their blocks
- **WHEN** the agent runs `c3po gates code architecture 6 7 8.3`
- **THEN** the blocks SHALL be `§6`, then `§7`, then `§8.3`, separated by one blank line
- **AND** the `§6` block SHALL list `§6.1` through `§6.13` under its body and the `§7` block SHALL list `§7.1` through `§7.5`
- **AND** the `§8.3` block SHALL be `§8.3` and `Mention it in the summary of the change.` with no subsection line

### Requirement: Gates help names the four probes
`c3po gates help`, `c3po gates --help`, and `c3po gates -h` SHALL print help and SHALL exit 0. `c3po help -f gates` SHALL print the same help. The help SHALL show the four probes `c3po gates`, `c3po gates <type>`, `c3po gates <type> <statute>`, and `c3po gates <type> <statute> <section>...`. It SHALL show that a section may be written `§1.1`, `1.1`, or `S.1.1`, and that several sections may follow the statute, with the example `c3po gates code architecture 6 7 8.3`. It SHALL say that a section with subsections lists those subsections under the body. It SHALL name `code` and `essays`, the statute `architecture`, and the 300-character limit. It SHALL NOT name `list`, `get`, or a `C/A/` id. The help SHALL say that architecture is stricter than style, framework choice, or micro-optimization. The help SHALL give the sequential rule: gates run in the list order, a failed gate fails every later gate, and those later gates are not judged. The help SHALL give the report line `§1  pass` or `§1  fail`, and after a fail `§1.1  fail  not judged`. The help SHALL say that the agent judges and c3po only prints the text, that there is no way to edit a gate, and that there is no gates screen. `c3po gates version` SHALL print `c3po <version>` and SHALL exit 0.

#### Scenario: Help
- **WHEN** the agent runs `c3po gates help`
- **THEN** the process SHALL exit 0
- **AND** the text SHALL show `c3po gates <type> <statute> <section>...`, `c3po gates code architecture §1.1`, and `c3po gates code architecture 6 7 8.3`
- **AND** the text SHALL show `§1.1`, `1.1`, and `S.1.1` as one section
- **AND** the text SHALL say a section with subsections lists those subsections under the body
- **AND** the text SHALL name `architecture`, `300`, `§1  pass`, and `not judged`

#### Scenario: Version
- **WHEN** the agent runs `c3po gates version`
- **THEN** stdout SHALL start with `c3po `
