# gmail Specification

## Purpose
c3po gmail tidy deletes mail that matches each preset spam list, shows one progress bar for the run, and prints a table of messages trashed per preset. c3po gmail tidy audit lists frequent unread senders that are not already on that list.
## Requirements
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

### Requirement: Tidy shows one progress bar
`c3po gmail tidy` SHALL draw one block progress bar for the whole run. The bar SHALL advance across every preset's sender groups. On a terminal, that bar SHALL update in place. The bar SHALL NOT name a preset or an email. When the run finishes, the bar line SHALL remain and SHALL show the bar and how many sender groups are done. Tidy SHALL then print a table with one row per preset. Each row SHALL name the preset, the email, and how many messages were trashed. The command SHALL NOT print one line per sender group and SHALL NOT print one bar per preset. The completion line SHALL remain `gmail tidy complete: presets=%d trashed=%d`. With no sender groups, tidy SHALL print no progress bar and SHALL print no table.

#### Scenario: A terminal redraws one line
- **WHEN** tidy cleans more than one preset on a terminal
- **THEN** the progress SHALL be redrawn on one line
- **AND** that line SHALL NOT name an email
- **AND** the output SHALL contain a table row for each preset

#### Scenario: A pipe does not print every group
- **WHEN** tidy cleans presets and stdout is not a terminal
- **THEN** the run SHALL produce one progress line
- **AND** the text SHALL NOT contain `processed group`
- **AND** the text SHALL contain a table header for preset, email, and trashed
- **AND** the text SHALL contain `gmail tidy complete: presets=`

