# gates Specification

## Purpose
TBD - created by archiving change add-gates. Update Purpose after archive.
## Requirements
### Requirement: Gates publish statutes of provisions
`c3po gates` SHALL print hardcoded legislation so an agent can judge work against it. c3po SHALL NOT search code for violations, call a model, or decide pass or fail. Only `tally` SHALL read a file or run git, and it SHALL read only the sheet, the paths its sites name, and `git status`. A type is a broad subject. The types SHALL be exactly `code` and then `essays`, in that order. A type SHALL hold zero or more statutes. `code` SHALL hold exactly the statute `architecture`. `essays` SHALL hold no statute. A statute SHALL hold provisions. A provision SHALL have a section, a heading of at most 40 characters, and a body of at most 300 Unicode code points. The legislation SHALL be compiled into the binary. The command SHALL NOT read a config file or a path under the user's home to find the text.

A published section SHALL be the section sign `§` (U+00A7) immediately followed by one or more positive integers separated by dots, with no space and no letter. `S.1`, `s.1`, `§ 1`, `§.1`, a bare `1`, `§0`, `§1.0`, and `§01` SHALL NOT be published sections. Provisions SHALL be ordered numerically, component by component, and a section that is a prefix of another SHALL come first. A provision SHALL be named by its type, its statute, and its section, for example `code architecture §1`. There SHALL be no combined id such as `C/A/S.1`.

A section argument SHALL name a published section when it is that section, that section without the section sign, or that section with `S.` in place of the section sign. `§6`, `6`, and `S.6` SHALL name `§6`. `§8.3`, `8.3`, and `S.8.3` SHALL name `§8.3`. The integers SHALL match exactly, so `6.0`, `01`, `0`, and `S.0` SHALL NOT name a section. `s.1`, `§ 1`, `§.1`, and `S§1` SHALL NOT name a section.

The `architecture` provisions SHALL be cut from the architecture guardrails text without adding or paraphrasing a rule, with markdown stripped, one bullet or one short paragraph per body. `§1`, `§2`, and `§3` of `architecture` SHALL be `Data model`, `State ownership`, and `Boundaries`, with the bodies `Everything else inherits its shape.`, `Each fact has exactly one owner and one write path.`, and `The core domain is independent of UI, database, and frameworks.` Their subsections SHALL be the goal paragraph, then each hard rule, then each default, then each red flag. `§4` through `§8` SHALL be how to apply, severity, checklist, reporting, and exceptions. The hard-rules severity paragraph SHALL be two provisions under `§5`. Subsections SHALL be numbered from 1 without gaps.

#### Scenario: Sections sort by number
- **GIVEN** provisions with sections `§10`, `§3.5.1`, `§2.3`, `§3`, `§1`, and `§2`
- **WHEN** they are put in section order
- **THEN** the order SHALL be `§1`, `§2`, `§2.3`, `§3`, `§3.5.1`, `§10`

#### Scenario: Only the section sign form parses
- **WHEN** `§1`, `§2.3`, `§3.5.1`, and `§10` are parsed as published sections
- **THEN** each SHALL be a section
- **AND** `S.1`, `s.1`, `§ 1`, `§.1`, `1`, `§0`, `§1.0`, and `§01` SHALL NOT be published sections

#### Scenario: Three forms name one section
- **WHEN** the section arguments `§6`, `6`, and `S.6` are resolved
- **THEN** each SHALL name `§6`
- **AND** `§8.3`, `8.3`, and `S.8.3` SHALL each name `§8.3`

#### Scenario: Every body fits
- **WHEN** every published body is counted with `utf8.RuneCountInString`
- **THEN** no body SHALL be over 300
- **AND** some body SHALL contain `stable, immutable identifier`, `exactly one write path`, `Dependencies MUST point inward`, and `inherits its shape`

### Requirement: Gates refuse unknown names and cannot be rewritten
`list`, `get`, `show`, `edit`, `add`, and `remove` SHALL NOT be commands. `c3po gates list`, `get`, `show`, `edit`, `add`, and `remove`, with or without more words, SHALL exit 2, and stderr SHALL say `unknown command` and that the gates change only when c3po itself is changed and reinstalled. Any other first word that is not a type, help, or version SHALL exit 2 and stderr SHALL say `unknown type`. An unknown statute SHALL exit 2 and stderr SHALL say `unknown statute`. A section argument that does not name a section published in that statute SHALL exit 2 and stderr SHALL say `unknown section` and quote that argument; `§9`, `9`, `s.1`, `§ 1`, `§.1`, `§0`, `0`, `S.0`, `§1.0`, `6.0`, `§01`, `01`, and `C/A/S.1` SHALL be unknown sections of `code architecture`. When several sections are requested and any one is unknown, the process SHALL exit 2, SHALL name that section as an unknown section, and SHALL NOT print the sections that were valid. A refusal SHALL print nothing on stdout.

#### Scenario: Unknown names
- **WHEN** the agent runs `c3po gates poems`, `c3po gates code style`, or `c3po gates code architecture §9`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL contain `unknown type`, `unknown statute`, or `unknown section` in turn
- **AND** stdout SHALL be empty

#### Scenario: The old id is not a section
- **WHEN** the agent runs `c3po gates code architecture C/A/S.1`, `6.0`, `01`, `0`, `S.0`, `s.1`, `§ 1`, or `§.1`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL contain `unknown section`
- **AND** stdout SHALL be empty

#### Scenario: One unknown section refuses the pass
- **WHEN** the agent runs `c3po gates code architecture 6 9 7`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL contain `unknown section "9"`
- **AND** stdout SHALL be empty

#### Scenario: A fourth argument
- **WHEN** the agent runs `c3po gates code architecture §1 x`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL contain `unknown section "x"`
- **AND** stdout SHALL be empty

#### Scenario: List and get are gone
- **WHEN** the agent runs `c3po gates list code` or `c3po gates get C/A/S.1`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL say `unknown command`
- **AND** stdout SHALL be empty

#### Scenario: No show or edit command
- **WHEN** the agent runs `c3po gates show code` or `c3po gates edit §1`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL say the text changes only when c3po is changed and reinstalled
- **AND** stdout SHALL be empty

### Requirement: Gates probe one level per argument
Each added argument after the type and statute SHALL deepen the same probe. `c3po gates` SHALL print `code` and `essays`, one per line, and SHALL exit 0. `c3po gates <type>` SHALL print the statute names of that type, one per line, and SHALL exit 0; a type with no statute SHALL print `no statutes`. `c3po gates <type> <statute>` SHALL print every provision of that statute in section order, one line each, as the section, two spaces, and the heading, with no banner, no receipt, and no blank line, and SHALL exit 0. The section SHALL be the published section such as `§1` or `§1.1`, so an agent can copy it as the next argument.

Every printed body SHALL be followed on the next line by `receipt ` and 8 lowercase hex characters. The hex SHALL be the first 8 characters of the SHA-256 digest, from Go `crypto/sha256`, of `version.String()`, a newline, the canonical section such as `§1.2`, a newline, and the body. A receipt SHALL NOT be stored in source. A title line, which is a section, two spaces, and a heading, SHALL NOT carry a receipt.

An immediate subsection of a section SHALL be a published section of the same statute that is that section, a dot, and one more positive integer. `§8.1` SHALL be an immediate subsection of `§8`. `§8.1.1` SHALL be an immediate subsection of `§8.1` and SHALL NOT be an immediate subsection of `§8`. A section with no immediate subsection is a leaf.

`c3po gates <type> <statute> <section>` SHALL exit 0 whichever form the section is written in. For a leaf it SHALL print only that provision's body and its receipt line. For a section with immediate subsections it SHALL print the body, the receipt line, a blank line, one title line per immediate subsection in section order, and then a closing line, with no banner and no trailing blank line. It SHALL NOT print a subsection's body or any deeper section. It SHALL NOT print the requested section, its heading, or the judging rule.

The closing line SHALL count only the immediate subsections whose bodies are not printed anywhere in the same output. It SHALL be `<n> bodies not shown: c3po gates <type> <statute> <args>`, or `1 body not shown: ...` when one is missing. The args SHALL be `<parent>.{1..<n>}`, with the parent written without the section sign, when the missing subsections are exactly `<parent>.1` through `<parent>.<n>` and `<n>` is at least 2. Otherwise the args SHALL be the missing sections without the section sign, in section order, separated by single spaces. When every immediate subsection's body is in the same output, there SHALL be no closing line.

`c3po gates <type> <statute> <section> <section>...` SHALL print every requested provision in one pass, in the order written and not in statute order, and SHALL exit 0. Each block SHALL be the published section on its own line, then the body, then the receipt line, then, when the section has immediate subsections, a blank line, one title line each, and the closing line when one is needed. A leaf block SHALL have no title lines. A blank line SHALL separate one block from the next. The last line SHALL be followed by one newline and no blank line. No heading and no probe output SHALL contain the judging rule. The command SHALL NOT refuse a batch for its size or order.

`c3po gates <type> <statute> all` SHALL print every provision of the statute in section order, each as the published section, the body, and the receipt line, with one blank line between provisions, and SHALL exit 0. It SHALL print no title line and no closing line. `all` SHALL NOT be a section: `all` with any other section argument, and `§all`, SHALL be an unknown section.

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
- **AND** stdout SHALL have one line per provision, no receipt, and no blank line

#### Scenario: One provision
- **WHEN** the agent runs `c3po gates code architecture §8.3`, `8.3`, or `S.8.3`
- **THEN** stdout SHALL be `Mention it in the summary of the change.`, a newline, `receipt `, the first 8 hex characters of SHA-256 over `version.String()`, `\n§8.3\n`, and that body, and a newline
- **AND** the process SHALL exit 0

#### Scenario: A provision with subsections
- **WHEN** the agent runs `c3po gates code architecture 8`, `§8`, or `S.8`
- **THEN** stdout SHALL be `A rule may be broken deliberately, never silently.`, the `§8` receipt line, a blank line, `§8.1  Comment at the site`, `§8.2  Contain the violation`, `§8.3  Mention it in the summary`, `§8.4  Legitimate exceptions`, and `4 bodies not shown: c3po gates code architecture 8.{1..4}`, each followed by a newline
- **AND** stdout SHALL NOT contain the body of `§8.1`
- **AND** the process SHALL exit 0

#### Scenario: Subsections list headings, not bodies
- **WHEN** the agent runs `c3po gates code architecture 1`
- **THEN** stdout SHALL contain the line `§1.1  Domain truth`
- **AND** stdout SHALL end with `19 bodies not shown: c3po gates code architecture 1.{1..19}`
- **AND** stdout SHALL NOT contain the body of `§1.1`

#### Scenario: Only the next level is listed
- **GIVEN** a statute with `§8`, `§8.1`, `§8.1.1` through `§8.1.6`, `§8.1.2.3`, and `§8.2`
- **WHEN** `§8` is printed
- **THEN** the title lines SHALL be `§8.1` and `§8.2` only
- **AND** stdout SHALL NOT contain `§8.1.1` or `§8.1.2.3`
- **WHEN** `§8.1` is printed in an output that also prints `§8.1.2`
- **THEN** the title lines SHALL be `§8.1.1` through `§8.1.6` only
- **AND** the closing line SHALL be `5 bodies not shown: c3po gates code deep 8.1.1 8.1.3 8.1.4 8.1.5 8.1.6`

#### Scenario: Several provisions in the order written
- **WHEN** the agent runs `c3po gates code architecture 8.3 8`
- **THEN** stdout SHALL be `§8.3`, `Mention it in the summary of the change.`, the `§8.3` receipt line, a blank line, `§8`, `A rule may be broken deliberately, never silently.`, the `§8` receipt line, a blank line, `§8.1  Comment at the site`, `§8.2  Contain the violation`, `§8.3  Mention it in the summary`, `§8.4  Legitimate exceptions`, and `3 bodies not shown: c3po gates code architecture 8.1 8.2 8.4`, each followed by a newline
- **AND** stdout SHALL NOT end with a blank line
- **AND** the process SHALL exit 0

#### Scenario: Shown subsections are not counted
- **WHEN** the agent runs `c3po gates code architecture 8 8.1 8.2 8.3 8.4`
- **THEN** stdout SHALL NOT contain `not shown`
- **WHEN** the agent runs `c3po gates code architecture 8 8.4 8.3`
- **THEN** the closing line SHALL be `2 bodies not shown: c3po gates code architecture 8.{1..2}`

#### Scenario: Several provisions keep their blocks
- **WHEN** the agent runs `c3po gates code architecture 6 7 8.3`
- **THEN** the blocks SHALL be `§6`, then `§7`, then `§8.3`, separated by one blank line
- **AND** the `§6` block SHALL list `§6.1` through `§6.13` and end with `13 bodies not shown: c3po gates code architecture 6.{1..13}`
- **AND** the `§7` block SHALL list `§7.1` through `§7.5` and end with `5 bodies not shown: c3po gates code architecture 7.{1..5}`
- **AND** the `§8.3` block SHALL be `§8.3`, `Mention it in the summary of the change.`, and its receipt line, with no title line

#### Scenario: All provisions
- **WHEN** the agent runs `c3po gates code architecture all`
- **THEN** stdout SHALL be 93 blocks in section order, each the section, the body, and the receipt line, separated by one blank line
- **AND** stdout SHALL contain `§8.3`, `Mention it in the summary of the change.`, and its receipt line
- **AND** stdout SHALL NOT contain a title line or `not shown`
- **AND** the process SHALL exit 0

#### Scenario: All is not a section
- **WHEN** the agent runs `c3po gates code architecture all 8` or `c3po gates code architecture 8 all`
- **THEN** the process SHALL exit 2
- **AND** stderr SHALL contain `unknown section`
- **AND** stdout SHALL be empty

### Requirement: Gates help names the four probes
`c3po gates help`, `c3po gates --help`, and `c3po gates -h` SHALL print help and SHALL exit 0. `c3po help -f gates` SHALL print the same help. The help SHALL show the four probes `c3po gates`, `c3po gates <type>`, `c3po gates <type> <statute>`, and `c3po gates <type> <statute> <section>...`, and SHALL show `c3po gates <type> <statute> all` and `c3po gates <type> <statute> tally <sheet>`. It SHALL show that a section may be written `§1.1`, `1.1`, or `S.1.1`, and that several sections may follow the statute, with the example `c3po gates code architecture 6 7 8.3`. It SHALL say that a section with subsections lists those subsections under the body and show the closing line `4 bodies not shown: c3po gates code architecture 8.{1..4}`. It SHALL show the receipt line `receipt <8 hex>` and say how the receipt is made. It SHALL show the sheet line `§1.2 <receipt> <pass|fail> <site> [files <path>...]` and one example sheet line whose receipt is the real receipt of that section in the running build. It SHALL name `code` and `essays`, the statute `architecture`, and the 300-character limit. It SHALL NOT name `list`, `get`, or a `C/A/` id. The help SHALL say that architecture is stricter than style, framework choice, or micro-optimization. The help SHALL give the judging rule: judge every provision; the first failure is the headline, and later fixes wait until that provision passes; every body is still judged. The help SHALL NOT say that a failed gate leaves later gates not judged. The help SHALL say that the agent judges and c3po only prints the text, that there is no way to edit a gate, and that there is no gates screen. `c3po gates version` SHALL print `c3po <version>` and SHALL exit 0.

#### Scenario: Help
- **WHEN** the agent runs `c3po gates help`
- **THEN** the process SHALL exit 0
- **AND** the text SHALL show `c3po gates <type> <statute> <section>...`, `c3po gates <type> <statute> all`, `c3po gates <type> <statute> tally <sheet>`, `c3po gates code architecture §1.1`, and `c3po gates code architecture 6 7 8.3`
- **AND** the text SHALL show `§1.1`, `1.1`, and `S.1.1` as one section
- **AND** the text SHALL say a section with subsections lists those subsections under the body
- **AND** the text SHALL show `receipt <8 hex>` and `§1.2 <receipt> <pass|fail> <site> [files <path>...]`
- **AND** the text SHALL name `architecture`, `300`, and `Every body is still judged`
- **AND** the text SHALL NOT contain `not judged`

#### Scenario: Version
- **WHEN** the agent runs `c3po gates version`
- **THEN** stdout SHALL start with `c3po `

### Requirement: Gates tally a sheet
`c3po gates <type> <statute> tally <sheet>` SHALL read the sheet file and count it against the statute. The command SHALL stay stateless, SHALL only print, and SHALL NOT decide whether the work passes. A sheet SHALL hold one line per provision in any order: a section in any accepted form, the receipt hex, `pass` or `fail`, a site, and optionally `files` followed by one or more paths. A site SHALL be `none` or `path:line`, where the path is relative to the current directory, exists, and has that line. `none` means the search found nothing, and a fail MAY use `none`. A blank line SHALL be skipped. A line with an unknown section, a receipt that does not match this build, a verdict other than `pass` or `fail`, no site, a site that is not `none` or a real `path:line`, extra words, or `files` with no path SHALL be a bad line. A second line for a section SHALL be a bad line.

The first line of stdout SHALL be `<n> of <total> judged`, where `<n>` counts the sections with a good line. Then, only when they apply, it SHALL print `missing` and the missing sections, then `c3po gates <type> <statute>` and those sections without the section sign; `unfixed` and the sections of fail lines that name no files; `unclaimed` and the changed files no good line names under `files`; and one line per bad line as `bad line <number>: <reason>: <line>`, where a bad path is named in the reason.

Changed files SHALL come from `git status --porcelain` run in the current directory, as paths relative to the current directory. Untracked files SHALL be ignored. A modified, deleted, or renamed path SHALL count, and a rename SHALL count as its new path.

The process SHALL exit 0 when every provision has a good line, there is no bad line, and every changed file is claimed. A fail that names no files SHALL be printed as `unfixed` and SHALL NOT by itself exit 2 in a clean tree. In a tree with any changed file, an unfixed fail SHALL exit 2. Every other sheet SHALL exit 2. A sheet that cannot be read SHALL exit 2 and stderr SHALL name the sheet. When git fails, the process SHALL exit 2 and stderr SHALL say `git status failed`. `tally` with no sheet or more than one SHALL exit 2.

#### Scenario: A complete sheet
- **GIVEN** a clean git repository and a sheet with one good pass line per provision
- **WHEN** the agent runs `c3po gates code architecture tally sheet.txt`
- **THEN** stdout SHALL be `93 of 93 judged`
- **AND** the process SHALL exit 0

#### Scenario: Missing sections
- **GIVEN** a sheet without `§1.3`, `§2.4`, and `§8`
- **WHEN** the agent runs tally
- **THEN** stdout SHALL be `90 of 93 judged`, `missing §1.3 §2.4 §8`, and `c3po gates code architecture 1.3 2.4 8`
- **AND** the process SHALL exit 2

#### Scenario: A bad receipt
- **GIVEN** a full sheet whose third line has a receipt that is not the `§1.2` receipt of this build
- **WHEN** the agent runs tally
- **THEN** stdout SHALL contain `bad line 3: bad receipt:` and that line
- **AND** stdout SHALL list `§1.2` as missing
- **AND** the process SHALL exit 2

#### Scenario: A missing path
- **GIVEN** a sheet line whose site is `nope.go:1` and no file `nope.go`
- **WHEN** the agent runs tally
- **THEN** stdout SHALL contain `missing path nope.go`
- **AND** the process SHALL exit 2

#### Scenario: An unclaimed changed file
- **GIVEN** a git repository with a modified tracked file `a.go` and a full sheet that names no files
- **WHEN** the agent runs tally
- **THEN** stdout SHALL be `93 of 93 judged` and `unclaimed a.go`
- **AND** the process SHALL exit 2
- **WHEN** one fail line names `files a.go`
- **THEN** the process SHALL exit 0

#### Scenario: An unfixed fail
- **GIVEN** a full sheet with a fail line for `§3.2` that names no files
- **WHEN** the agent runs tally in a clean tree
- **THEN** stdout SHALL contain `unfixed §3.2`
- **AND** the process SHALL exit 0
- **WHEN** the agent runs tally in a tree with a changed file
- **THEN** the process SHALL exit 2

#### Scenario: No git
- **WHEN** the agent runs tally outside a git repository
- **THEN** stderr SHALL say `git status failed`
- **AND** the process SHALL exit 2

