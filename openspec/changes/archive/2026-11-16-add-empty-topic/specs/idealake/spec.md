## MODIFIED Requirements

### Requirement: An idea is one short line in a topic
`c3po idealake add <topic> [<idea>...]` SHALL store each given idea under that topic. With no idea arguments on a terminal, or with no idea arguments and no non-empty stdin lines, the command SHALL store that topic, SHALL store no idea, and SHALL exit 0. Storing that same topic again SHALL leave it in place and SHALL exit 0. After trimming, an idea SHALL be one line of at most 300 characters and a topic SHALL be one line of at most 80 characters. The character count SHALL be Unicode code points. An empty idea, an empty topic, a newline inside either, or a value over its limit SHALL exit 2 and SHALL name the limit when the value is over it. When any idea in the call is rejected, the command SHALL store none of the ideas in that call and SHALL NOT create that topic. Each stored idea SHALL receive a new decimal id, and an id SHALL NOT be reused. The text output of stored ideas SHALL print each stored id, topic, and text. The text output of a topic stored with no new ideas SHALL print the count, two spaces, and the topic. When no idea arguments are given and stdin is not a terminal, each non-empty line SHALL be one idea. `c3po idealake --json add` of one or more ideas SHALL print a JSON object with an `ideas` array and SHALL NOT print an embedding vector. `c3po idealake --json add` of a topic with no ideas SHALL print a JSON object with a `topics` array naming that topic and its count, and SHALL NOT print an embedding vector.

#### Scenario: Chosen ideas are stored under the topic
- **WHEN** the operator runs `c3po idealake add xyz "one" "two"`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL show two new ids
- **AND** `c3po idealake list xyz` SHALL show both texts and SHALL NOT show an idea from another topic

#### Scenario: A long idea stores nothing
- **WHEN** one idea in the call is 301 characters and another idea is within the limit
- **THEN** the command SHALL exit 2
- **AND** no idea from that call SHALL be stored
- **AND** that call SHALL NOT create the topic

#### Scenario: Stdin supplies the ideas
- **WHEN** stdin is not a terminal and the operator runs `c3po idealake add xyz` with one idea on each line
- **THEN** each non-empty line SHALL be stored
- **AND** a blank line SHALL be skipped

#### Scenario: A topic is stored with no ideas
- **WHEN** the operator runs `c3po idealake add xyz` with no idea arguments
- **THEN** the command SHALL exit 0
- **AND** `c3po idealake topics` SHALL list xyz with a count of 0
- **AND** `c3po idealake list xyz` SHALL print `no ideas`

#### Scenario: The same topic is stored again
- **WHEN** the operator runs `c3po idealake add xyz` twice with no ideas
- **THEN** `c3po idealake topics` SHALL list xyz once

### Requirement: Agents can list, show, and remove ideas
`c3po idealake list` SHALL print every idea. `c3po idealake list <topic>` SHALL print that topic only. An empty result SHALL print `no ideas` and SHALL exit 0. `c3po idealake show <id>` SHALL print that idea, its topic, and its created time. A missing id SHALL exit 1 and the error SHALL contain `no idea`. `c3po idealake remove <id>` SHALL delete that idea and SHALL print `removed` and the id. A missing id SHALL exit 1. `c3po idealake topics` SHALL print each topic with its count, including a topic that has no ideas. An empty lake SHALL print `no topics` and SHALL exit 0. `--json` before `list`, `show`, `remove`, or `topics` SHALL print JSON and SHALL NOT include an embedding vector. Extra arguments SHALL exit 2.

#### Scenario: Remove deletes one idea
- **WHEN** the operator stores an idea and then removes its id
- **THEN** the command SHALL exit 0
- **AND** a later show of that id SHALL exit 1
- **AND** the next stored idea SHALL receive a new id

#### Scenario: JSON list omits vectors
- **WHEN** the operator runs `c3po idealake --json list`
- **THEN** the command SHALL exit 0
- **AND** the JSON SHALL name each idea id, topic, and text
- **AND** the JSON SHALL NOT contain an embedding vector

#### Scenario: Topics lists an empty topic
- **WHEN** one topic was stored with no ideas and another topic has one idea
- **THEN** `c3po idealake topics` SHALL print both
- **AND** the empty topic SHALL show a count of 0

### Requirement: Records use the c3po sqlite backup
Each idea SHALL be stored in the sqlx database `c3po` as a record with app `idealake` and kind `idea`. A topic with no ideas SHALL be stored in that same database as a record with app `idealake` and kind `topic`. A write SHALL publish that sqlite file with nebula s3 the same way other c3po records do. The command SHALL NOT call nebula sql.

#### Scenario: An idea is a c3po record
- **WHEN** an idea is stored
- **THEN** the c3po sqlite file SHALL contain that idea under app `idealake`

#### Scenario: An empty topic is a c3po record
- **WHEN** a topic is stored with no ideas
- **THEN** the c3po sqlite file SHALL contain that topic under app `idealake` and kind `topic`

### Requirement: Help names the limit
`c3po idealake` with no arguments and `c3po idealake help` SHALL exit 0 and SHALL name 300 characters, a topic, `search`, `contains`, `XAI_API_KEY`, and `--json`. The text SHALL show that add can store a topic with no ideas. `c3po idealake version` SHALL print the c3po version. An unknown subcommand SHALL exit 2 and SHALL print the help. `c3po help -f idealake` SHALL print the same text as `c3po idealake help`.

#### Scenario: Help names the limit
- **WHEN** the operator runs `c3po idealake help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL include `300`
- **AND** the text SHALL include `XAI_API_KEY`
- **AND** the text SHALL show that a topic can be stored with no ideas
