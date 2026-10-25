## REMOVED Requirements

### Requirement: Tidy classifies unread mail with grok
**Reason**: Tidy deletes mail that matches the configured spam list. It does not classify mail or add senders.
**Migration**: Keep senders on the preset spam list. Run `c3po gmail tidy` to delete that mail. Run `c3po gmail tidy audit` to see frequent unread senders.

## ADDED Requirements

### Requirement: Tidy deletes configured spam
`c3po gmail tidy` SHALL trash mail that matches each preset's spam senders, minus that preset's spam excludes. It SHALL NOT add a sender, SHALL NOT hardcode a sender, and SHALL NOT classify mail. It SHALL NOT mark mail read. The spam list SHALL stay unchanged. With no spam senders left after excludes, it SHALL trash nothing and SHALL exit 0. The command SHALL print `gmail tidy complete: presets=%d trashed=%d`. Spam cleanup SHALL query only the configured senders. It SHALL NOT query a domain the preset list omits.

#### Scenario: Tidy trashes configured spam
- **WHEN** a preset lists `spam@evil.test` and that sender has a message
- **THEN** that message SHALL be trashed
- **AND** the spam list SHALL stay unchanged
- **AND** the text SHALL contain `gmail tidy complete: presets=`

#### Scenario: An unlisted sender stays
- **WHEN** unread mail is from a sender that is not on the spam list
- **THEN** that mail SHALL stay untrashed
- **AND** cleanup SHALL NOT query a ClickUp domain that the list omits

### Requirement: Tidy audit lists frequent unread senders
`c3po gmail tidy audit` SHALL list unread senders for every preset from `is:unread -in:sent`. The list SHALL be the highest unread count first. Equal counts SHALL be ordered by sender. A sender already covered by that preset's spam list or spam excludes SHALL be omitted. An empty From SHALL be omitted. The command SHALL NOT trash mail and SHALL NOT change the spam list. Each row SHALL name the preset, the sender, and the unread count. The command SHALL print `gmail tidy audit complete: presets=%d senders=%d`.

#### Scenario: The busiest unread sender is first
- **WHEN** one preset has three unread messages from `a@bulk.test` and one from `b@bulk.test`
- **THEN** `a@bulk.test` SHALL appear before `b@bulk.test`
- **AND** the row SHALL show unread 3 for `a@bulk.test`
- **AND** no mail SHALL be trashed

#### Scenario: A listed spam sender is omitted
- **WHEN** the busiest unread sender is already on the spam list
- **THEN** that sender SHALL NOT appear in the audit
