## ADDED Requirements

### Requirement: A record body does not repeat its key
The record key of an idea SHALL be its id, and the record key of a topic SHALL be its name. An idea body SHALL NOT hold `id`, and a topic body SHALL NOT hold `topic`. Loading an idea SHALL take its id from the record key, and listing topics SHALL take each name from the record key. Before it lists ideas or topics, idealake SHALL run the migration `idealake-body-key-1` once, which SHALL remove `id` from every idea body and `topic` from every topic body. A body that is not valid JSON SHALL be left unchanged. Command output SHALL NOT change.

#### Scenario: A new idea and topic store no key copy
- **WHEN** an idea is stored and a topic is stored with no ideas
- **THEN** the idea body SHALL NOT contain `id`
- **AND** the topic body SHALL NOT contain `topic`

#### Scenario: An old body loses its copy and still loads
- **GIVEN** idea `7` whose body holds `"id":"7"` and topic `xyz` whose body holds `"topic":"xyz"`
- **WHEN** `c3po idealake show 7` and `c3po idealake topics` run
- **THEN** each SHALL exit 0 and print that idea and that topic
- **AND** the idea body SHALL NOT contain `id` and SHALL still name its topic
- **AND** the topic body SHALL NOT contain `topic`
