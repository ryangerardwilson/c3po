## Context

Ideas are records with app `idealake` and kind `idea`. Topics are counted from those rows, so a topic with no ideas cannot be listed.

## Goals / Non-Goals

- Goals: store a named topic before any idea exists, list it with count 0, and keep later idea adds unchanged.
- Non-Goals: a screen, a command that deletes a topic, or a change to search ranking.

## Decisions

- `c3po idealake add <topic>` with no idea arguments, and the same command with empty non-terminal stdin, writes one record with kind `topic`. The key is the trimmed topic.
- The same topic written again is left in place.
- `topics` unions kind `topic` rows with topics that appear on ideas. A topic that only appears on ideas still disappears when its last idea is removed.
- Text output for a topic-only add is the count, two spaces, and the topic, the same shape as `topics`.
- JSON for a topic-only add is `{"topics":[{"topic":"...","n":0}]}`.

## Risks / Trade-offs

- An empty topic stays listed. There is no delete-topic command in this change.
- Agents that treated exit 2 as "an idea is required" now create a topic instead.

## Migration Plan

- No stored idea rows move. New topic rows use kind `topic`.

## Open Questions

- None.
