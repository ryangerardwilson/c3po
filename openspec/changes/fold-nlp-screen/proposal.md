# Fold the nlp trainer into a screen

## Why
The trainer is a separate program. It should be one c3po screen, next to the other terminal screens.

## What Changes
- `c3po tui nlp` opens the trainer.
- `c3po nlp` is not a command.
- Courses stay in the sqlx database `nlp`.
- The screen stores its place in the sqlx database `c3po` and does not store course markdown there.

## Impact
Specs: `openspec/specs/nlp/spec.md`, `openspec/specs/scout/spec.md`, `openspec/specs/tui/spec.md`. Code: `nlp/`, `internal/scout`. Rollback is the previous c3po binary. Existing course rows stay in the `nlp` database.
