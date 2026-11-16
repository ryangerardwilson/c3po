# idealake Specification

## Purpose
TBD - created by archiving change add-idealake. Update Purpose after archive.
## Requirements
### Requirement: An idea is one short line in a topic
`c3po idealake add <topic> <idea> [<idea>...]` SHALL store each idea under that topic. After trimming, an idea SHALL be one line of at most 300 characters and a topic SHALL be one line of at most 80 characters. The character count SHALL be Unicode code points. An empty idea, an empty topic, a newline inside either, or a value over its limit SHALL exit 2 and SHALL name the limit when the value is over it. When any idea in the call is rejected, the command SHALL store none of the ideas in that call. Each stored idea SHALL receive a new decimal id, and an id SHALL NOT be reused. The text output SHALL print each stored id, topic, and text. When no idea arguments are given and stdin is not a terminal, each non-empty line SHALL be one idea. A terminal with no idea arguments SHALL exit 2. `c3po idealake --json add` SHALL print a JSON object with an `ideas` array and SHALL NOT print an embedding vector.

#### Scenario: Chosen ideas are stored under the topic
- **WHEN** the operator runs `c3po idealake add xyz "one" "two"`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL show two new ids
- **AND** `c3po idealake list xyz` SHALL show both texts and SHALL NOT show an idea from another topic

#### Scenario: A long idea stores nothing
- **WHEN** one idea in the call is 301 characters and another idea is within the limit
- **THEN** the command SHALL exit 2
- **AND** no idea from that call SHALL be stored

#### Scenario: Stdin supplies the ideas
- **WHEN** stdin is not a terminal and the operator runs `c3po idealake add xyz` with one idea on each line
- **THEN** each non-empty line SHALL be stored
- **AND** a blank line SHALL be skipped

### Requirement: Agents can list, show, and remove ideas
`c3po idealake list` SHALL print every idea. `c3po idealake list <topic>` SHALL print that topic only. An empty result SHALL print `no ideas` and SHALL exit 0. `c3po idealake show <id>` SHALL print that idea, its topic, and its created time. A missing id SHALL exit 1 and the error SHALL contain `no idea`. `c3po idealake remove <id>` SHALL delete that idea and SHALL print `removed` and the id. A missing id SHALL exit 1. `c3po idealake topics` SHALL print each topic with its count. An empty lake SHALL print `no topics` and SHALL exit 0. `--json` before `list`, `show`, `remove`, or `topics` SHALL print JSON and SHALL NOT include an embedding vector. Extra arguments SHALL exit 2.

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

### Requirement: Search ranks ideas in a topic by meaning
`c3po idealake search <topic> <query>` SHALL embed `query: ` plus the query with the xAI embeddings API, model `v1`, and SHALL rank ideas in that topic whose stored vector was made from `passage: ` plus the idea text by that same model. The rank SHALL be cosine similarity, highest first, and SHALL include at most 20 ideas. An idea SHALL be able to rank first even when its text does not contain the query. The command SHALL NOT include an idea only because its text contains the query. The key SHALL be `XAI_API_KEY` and SHALL NOT be printed. When the topic has an embedded idea and the query cannot be embedded, the command SHALL exit 1, stderr SHALL contain `embedding unavailable`, and stdout SHALL NOT list ideas. When every idea in the topic has no vector, the command SHALL exit 0, SHALL print `no ideas`, and stderr SHALL say how many ideas have no embedding. A missing topic or an empty query SHALL exit 2. An idea SHALL still be stored when its own embedding cannot be made, and stderr SHALL contain `stored without an embedding`.

#### Scenario: A nearer idea ranks first without the query words
- **GIVEN** two ideas in one topic whose embeddings differ
- **AND** neither idea text contains the query
- **WHEN** the operator searches that topic for a query nearer to the second idea
- **THEN** that idea SHALL be printed before the other
- **AND** the command SHALL exit 0

#### Scenario: Search does not fall back to the idea text
- **WHEN** an idea in the topic contains the query and has no embedding
- **THEN** search SHALL exit 0
- **AND** stdout SHALL NOT include that idea
- **AND** stderr SHALL say that ideas have no embedding

#### Scenario: Search stays inside the topic
- **WHEN** a nearer idea is stored in another topic
- **THEN** a search of the first topic SHALL NOT print that idea

### Requirement: Contains matches lowercase text
`c3po idealake contains <text> [topic]` SHALL print every idea whose text contains that text, ignoring case. The optional topic SHALL limit the match to that topic. The command SHALL NOT call the embeddings API and SHALL NOT rank by similarity. An empty result SHALL print `no ideas` and SHALL exit 0. An empty text SHALL exit 2.

#### Scenario: Contains ignores case inside one topic
- **WHEN** topic xyz has an idea containing `Quiet` and another topic has an idea containing `quiet`
- **AND** the operator runs `c3po idealake contains quiet xyz`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL include the xyz idea
- **AND** the text SHALL NOT include the other topic

#### Scenario: Contains finds an idea search cannot rank
- **WHEN** an idea contains a word and has no embedding
- **THEN** `c3po idealake contains` of that word SHALL include the idea

### Requirement: Records use the c3po sqlite backup
Each idea SHALL be stored in the sqlx database `c3po` as a record with app `idealake` and kind `idea`. A write SHALL publish that sqlite file with nebula s3 the same way other c3po records do. The command SHALL NOT call nebula sql.

#### Scenario: An idea is a c3po record
- **WHEN** an idea is stored
- **THEN** the c3po sqlite file SHALL contain that idea under app `idealake`

### Requirement: Help names the limit
`c3po idealake` with no arguments and `c3po idealake help` SHALL exit 0 and SHALL name 300 characters, a topic, `search`, `contains`, `XAI_API_KEY`, and `--json`. `c3po idealake version` SHALL print the c3po version. An unknown subcommand SHALL exit 2 and SHALL print the help. `c3po help -f idealake` SHALL print the same text as `c3po idealake help`.

#### Scenario: Help names the limit
- **WHEN** the operator runs `c3po idealake help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL include `300`
- **AND** the text SHALL include `XAI_API_KEY`

