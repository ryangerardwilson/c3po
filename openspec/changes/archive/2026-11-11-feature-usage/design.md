## Context

c3po already stores tokens, Drive state, tunnel routes, and TUI sessions in the sqlx database `c3po`. Feature use is a different record: many small counts, including one for every `c3po ast hook` call.

## Goals / Non-Goals

**Goals:**

- Show catalog features that have never been used, so a deletion has a count behind it.
- Keep human, agent, and service counts apart.
- Keep a failed write from changing a command's exit code or stderr.

**Non-Goals:**

- An event row per invocation. `ast hook` runs on every tool call.
- Storing argv, paths, or secrets.
- Counting keys inside a TUI screen.
- A new peer app. `c3po usage` is a top-level command, like help and version.

## Decisions

The feature name comes from a fixed catalog. Help and version collapse to `help` and `version`. An unknown word becomes `unknown`, or the app name when that name is already in the catalog. A typed word never becomes a new feature.

The actor is `C3PO_ACTOR` when it is `human`, `agent`, or `service`. Otherwise an agent environment marker wins, then a terminal is a human, and everything else is a service.

One row is one feature, one actor, and one UTC day. `n` counts runs. `ok` counts exits of 0. The increment uses one immediate transaction.

`c3po usage` prints before that invocation is recorded, so the report does not include itself.

Tests that leave `SQLX_ROOT` unset do not write. `C3PO_SQLX=off` does not write.

## Risks / Trade-offs

- A command that is missing from the catalog shows under "outside the catalog" only after someone runs it. Until then it is invisible. The catalog has to be updated with the command.
- Counts start the day this binary is installed. Older use is not reconstructed.
