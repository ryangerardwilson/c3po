## Context

`c3po tui looptab` lists jobs from `looptab now` and runs one index. The scheduler file is `~/.config/looptab/looptab`. The live job `18c57fbc` is a `@grok` prompt that cleans Gmail. The Decisions API is `POST /v1/decisions` with model `gpt-6-luna`. 1Password item `WIOM_OPENAI_API_KEY` is in the Wiom vault.

## Goals / Non-Goals

- Register jobs files and edit them in vim, then install them onto the scheduler file.
- Keep the TUI as run-now only.
- Replace the Grok cleanup with `c3po gmail tidy`.
- Leave `gmail.timer` as the hourly spam clean.
- Leave the Looptab module unimported. The screen still shells out to the `looptab` binary.

## Decisions

- Registrations are one sqlx record, app `looptab`, kind `files`, item `files`, at `~/.local/share/c3po/looptab/files.json`. `C3PO_SQLX=off` keeps that JSON as a file.
- The scheduler file is generated output. Registering that path is rejected so a reinstall cannot duplicate it.
- An empty catalog still writes the scheduler file, with the install header and no jobs.
- vim is the editor. A missing vim says `vim is not on PATH` and returns.
- `c3po gmail tidy` asks one choice and one predicate per unread message, in batches of 150 messages. Trash and mark-read require confidence of at least 0.8. A disposable From address is spam-added only when its predicate is at least 0.9 and the choice is trash. Anything else stays unread.
- The key is `OPENAI_API_KEY` when set. Otherwise the command reads `WIOM_OPENAI_API_KEY` from the Wiom vault through c3po pass and does not print it. A missing key leaves unclassified mail in place.
- ClickUp domains `@clickup.com`, `@mail.clickup.com`, and `@tasks.clickup.com` are spam-added on every preset before the first spam clean, with or without a key.

## Risks / Trade-offs

- A Decisions refusal or a low confidence leaves the message unread.
- The request body is not logged. A non-200 response reports the status only.
