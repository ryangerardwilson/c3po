## Purpose

c3po gmail tidy classifies unread mail with the OpenAI Decisions API and applies the same cleanup the morning Looptab job used to ask of an agent.

## ADDED Requirements

### Requirement: Tidy classifies unread mail with the Decisions API
`c3po gmail tidy` SHALL add `@clickup.com`, `@mail.clickup.com`, and `@tasks.clickup.com` on every preset, then run spam clean. It SHALL list unread mail once per preset with `is:unread -in:sent`. With no unread mail, it SHALL succeed and SHALL leave the Decisions API uncalled. With unread mail, it SHALL classify each message with `POST https://api.openai.com/v1/decisions` and model `gpt-6-luna`. The key SHALL be `OPENAI_API_KEY` when that variable is set, and otherwise the 1Password item `WIOM_OPENAI_API_KEY` in the Wiom vault. The key SHALL stay off the command output. A missing key SHALL exit 1 and SHALL leave unclassified mail untrashed. Trash and mark-read SHALL happen only when the choice confidence is at least 0.8. A lower confidence SHALL leave the message unread. The From address SHALL be spam-added only when the disposable probability is at least 0.9 and the choice is trash. Disposable senders for one preset SHALL be added in one spam add. Kept automated mail SHALL be marked read. Direct human mail, security mail, and legal mail SHALL stay unread. The command SHALL print `gmail tidy complete: presets=%d unread=%d trashed=%d marked_read=%d spam_added=%d left=%d`.

#### Scenario: No unread mail skips classification
- **WHEN** the operator runs `c3po gmail tidy` and no preset has unread mail
- **THEN** the command SHALL exit 0
- **AND** the ClickUp domains SHALL be on every preset
- **AND** the Decisions API SHALL stay uncalled
- **AND** the text SHALL contain `gmail tidy complete: presets=`

#### Scenario: A missing key leaves unread mail in place
- **WHEN** unread mail is present and neither `OPENAI_API_KEY` nor the Wiom item is available
- **THEN** the command SHALL exit 1
- **AND** the text SHALL say `OPENAI_API_KEY is required to classify unread mail`
- **AND** that unread mail SHALL stay untrashed

#### Scenario: Confident trash and read are applied
- **WHEN** classification returns trash at confidence 0.8 or more, read at confidence 0.8 or more, and a lower confidence
- **THEN** the trash message SHALL be deleted by id
- **AND** the read message SHALL be marked read
- **AND** the low-confidence message SHALL stay unread
- **AND** a disposable From on the trash message SHALL be spam-added
- **AND** the text SHALL contain `gmail tidy complete:`
