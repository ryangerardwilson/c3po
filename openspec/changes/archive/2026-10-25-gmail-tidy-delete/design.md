## Context

Each preset's spam senders and spam excludes are fields on the Gmail config file. sqlx database `c3po` stores tokens, Drive sync state, and tunnel routes. Operator config stays a file. Cleanup also appends `@clickup.com` and `@mail.clickup.com` even when the config omits them. Tidy then classifies remaining unread mail with grok.

## Goals / Non-Goals

- Goals: tidy trashes only the configured spam list. Audit prints unread senders by count so a sender can be added later. No hardcoded sender.
- Non-Goals: moving `spam_senders` into sqlx. Changing `spam add` or the hourly spam-clean timer.

## Decisions

- Tidy calls the same cleanup as spam clean, then prints `gmail tidy complete: presets=%d trashed=%d`.
- Cleanup iterates the configured senders only. Excludes still drop a sender.
- Audit lists `is:unread -in:sent`, skips an empty From, skips a sender already covered by the spam list or the excludes, and orders the rest by unread count, then sender. It does not trash mail or edit the list.

## Risks / Trade-offs

- A sender that was deleted only because of the hardcoded ClickUp domains stays until it is on the preset list. Those domains are already on the live presets.

## Migration Plan

Run `c3po gmail tidy` to delete configured spam. Run `c3po gmail tidy audit` to see frequent unread senders. Add one with `c3po gmail <preset> spam add`.

## Open Questions

None.
