## ADDED Requirements

### Requirement: The slack event cache stores each fact once
The slack event cache SHALL be a copy of Slack's answers, kept in `events-<preset>.db`. A conversation's `last_read` SHALL be stored only in its `last_read` column and SHALL NOT be kept in `info_json`; a `last_read` that Slack reports in a conversation's info SHALL be folded into that column. A message SHALL keep its ts only in `message_json`; `ts` and `sort_ts` SHALL be generated from it and SHALL NOT be stored. A message SHALL NOT store a copy of its user, its text, the event that delivered it, or an unread flag, and it SHALL be unique on channel and ts. A cached message SHALL be unread when its ts is after its conversation's `last_read` and it is not the operator's own. Marking a conversation read SHALL go through the one conversation write path, SHALL NOT move `last_read` back, and SHALL run only after Slack accepted the mark. Times SHALL be unix seconds in UTC. The last sync SHALL be one typed `sync_state` row, and the schema version SHALL be `PRAGMA user_version`. The events table SHALL be the one record of which event delivered which message. Each conversation and each message SHALL have one write path.

#### Scenario: last_read is stored once
- **WHEN** a conversation whose Slack info holds `last_read` is stored and then marked read
- **THEN** `last_read` SHALL hold the newer value
- **AND** `info_json` SHALL NOT contain `last_read`

### Requirement: The slack event cache schema is versioned and enforced
The cache SHALL record its schema version in `cache_state` and SHALL move an older cache forward in one transaction when it opens. A second process that waited SHALL see the new version and SHALL run nothing. A message and an event SHALL reference their conversation, and deleting a conversation SHALL delete them. The schema SHALL reject an unknown surface, a flag other than 0 or 1, and an unread mark at or before `last_read`. Moving a version 1 cache to version 2 SHALL keep every conversation, message, and event, SHALL give an orphaned message or event a placeholder conversation, SHALL take a missing payload ts from the old ts column, and SHALL fail without change when a reference would break. Deleting the cache file SHALL be the rollback, and the next sync SHALL refill it.

#### Scenario: A version 1 cache moves to version 2
- **GIVEN** a version 1 cache with a conversation whose `info_json` holds `last_read`, two messages, and an event for an unknown channel
- **WHEN** the cache opens
- **THEN** the version SHALL be 2
- **AND** `info_json` SHALL NOT contain `last_read`
- **AND** both messages SHALL load with their ts
- **AND** the unknown channel SHALL have a placeholder conversation

#### Scenario: A message needs its conversation
- **WHEN** a message is stored for a channel with no conversation row
- **THEN** the write SHALL fail
