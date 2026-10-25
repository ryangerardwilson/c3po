## Why

`c3po gmail tidy` hardcodes ClickUp domains and classifies unread mail. The spam list on each preset is already the list of senders to delete.

## What Changes

- `c3po gmail tidy` trashes mail that matches each preset's spam senders, minus that preset's spam excludes.
- It does not add a sender, hardcode a sender, classify mail, or mark mail read.
- `c3po gmail tidy audit` lists unread senders that are not already on that list, highest count first.
- Spam cleanup no longer inserts domains the preset list omits.

## Capabilities

### New Capabilities

### Modified Capabilities

- `gmail`: Tidy deletes configured spam, and tidy audit lists frequent unread senders.

## Impact

`c3po gmail tidy` and `c3po gmail spam clean`. The spam list stays `spam_senders` in the Gmail config. sqlx is not that list.
