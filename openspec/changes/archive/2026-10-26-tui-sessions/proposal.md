# TUI sessions

## Why

Tesseract can restore a c3po screen only when that screen has a session id and a workspace. The nine screens did not store that place.

## What Changes

- Each c3po TUI screen stores its place in the sqlx database `c3po` and accepts `--resume <id>`.
- Help and version do not open a session. A pipe with no resume id does not write one.
- Pass stores the vault and item id, not the secret.

## Impact

Capabilities: tui, scout. Migration is the first open of a screen on a terminal. Rollback is the previous binary. The session rows can stay in `c3po`.
