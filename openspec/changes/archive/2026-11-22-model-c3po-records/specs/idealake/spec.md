## MODIFIED Requirements

### Requirement: Records use the c3po sqlite backup
Each topic SHALL be a row in `idealake_topic`, unique by name. Each idea SHALL be a row in `idealake_idea` whose `topic_id` references its topic. A topic that still has ideas SHALL NOT be deleted. Storing an idea SHALL create its topic when it is missing, in the same transaction. A topic with no ideas SHALL be a topic row with no idea rows. The schema SHALL reject an empty, multi-line, or too-long topic or idea. An idea id SHALL NOT be issued twice, even after that idea is removed. A write SHALL publish the c3po sqlite file with nebula s3 the same way other c3po writes do. The command SHALL NOT call nebula sql.

#### Scenario: An idea is a c3po record
- **WHEN** an idea is stored in topic `xyz`
- **THEN** the c3po sqlite file SHALL hold that idea with a reference to topic `xyz`
- **AND** deleting topic `xyz` SHALL fail while the idea exists

#### Scenario: An empty topic is a c3po record
- **WHEN** a topic is stored with no ideas
- **THEN** the c3po sqlite file SHALL hold that topic and no idea for it

#### Scenario: A removed id is not issued again
- **GIVEN** idea `1` was stored and removed
- **WHEN** another idea is stored
- **THEN** its id SHALL be `2`

## REMOVED Requirements

### Requirement: A record body does not repeat its key
**Reason**: Ideas and topics are no longer JSON record bodies. The id and the name are columns, stored once.
**Migration**: Migration 8 copies every idea and topic record into the tables and removes the records.
