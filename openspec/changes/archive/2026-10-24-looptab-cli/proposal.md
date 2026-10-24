## Why

The only Looptab job spends a Grok session on Gmail cleanup that is a fixed sequence plus one classification step. Choosing and editing jobs is also stuck inside the run-now screen.

## What Changes

- `c3po looptab` registers jobs files, stores that catalog in the sqlx database `c3po`, opens a jobs file in vim, and installs the files onto the Looptab scheduler.
- `c3po tui looptab` stays the screen that runs any one job now.
- `c3po gmail tidy` adds the ClickUp spam domains, runs spam clean, classifies unread mail with the OpenAI Decisions API, trashes confident promo by message id, marks kept automated mail read, and spam-adds disposable senders. The API key is `OPENAI_API_KEY` or the Wiom vault item `WIOM_OPENAI_API_KEY`.

## Impact

- Specs: `openspec/specs/looptab/spec.md`, `openspec/specs/scout/spec.md`, `openspec/specs/gmail/spec.md`
- Code: `looptab/`, `gmail/`, `internal/scout`
