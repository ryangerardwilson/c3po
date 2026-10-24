# gmail Specification

## Purpose
c3po gmail tidy classifies unread mail with one grok CLI decision and applies that cleanup on every preset.

## Requirements

### Requirement: Tidy classifies unread mail with grok
`c3po gmail tidy` SHALL add `@clickup.com`, `@mail.clickup.com`, and `@tasks.clickup.com` on every preset, then run spam clean. It SHALL list unread mail once per preset with `is:unread -in:sent`. With no unread mail, it SHALL succeed and SHALL leave grok uncalled. With unread mail, it SHALL classify each message with one `grok` process per chunk of at most 150 messages. That process SHALL be a single turn (`--max-turns 1`) with `--json-schema`, and its working directory SHALL be an empty directory created for the call. The process SHALL NOT receive `--always-approve`. A missing `grok` binary SHALL exit 1, the text SHALL say `grok is required to classify unread mail`, and unclassified mail SHALL stay untrashed. A grok failure SHALL exit 1 and SHALL leave unclassified mail untrashed. Trash and mark-read SHALL happen only when the choice confidence is at least 0.8. A lower confidence SHALL leave the message unread. The From address SHALL be spam-added only when the disposable probability is at least 0.9 and the choice is trash. Disposable senders for one preset SHALL be added in one spam add. Kept automated mail SHALL be marked read. Direct human mail, security mail, and legal mail SHALL stay unread. The command SHALL print `gmail tidy complete: presets=%d unread=%d trashed=%d marked_read=%d spam_added=%d left=%d`.

#### Scenario: No unread mail skips classification
- **WHEN** the operator runs `c3po gmail tidy` and no preset has unread mail
- **THEN** the command SHALL exit 0
- **AND** the ClickUp domains SHALL be on every preset
- **AND** grok SHALL stay uncalled
- **AND** the text SHALL contain `gmail tidy complete: presets=`

#### Scenario: A missing grok binary leaves unread mail in place
- **WHEN** unread mail is present and `grok` is not available to the command
- **THEN** the command SHALL exit 1
- **AND** the text SHALL say `grok is required to classify unread mail`
- **AND** that unread mail SHALL stay untrashed

#### Scenario: Confident trash and read are applied
- **WHEN** classification returns trash at confidence 0.8 or more, read at confidence 0.8 or more, and a lower confidence
- **THEN** the trash message SHALL be deleted by id
- **AND** the read message SHALL be marked read
- **AND** the low-confidence message SHALL stay unread
- **AND** a disposable From on the trash message SHALL be spam-added
- **AND** the text SHALL contain `gmail tidy complete:`
