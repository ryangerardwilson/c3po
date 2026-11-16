## Why

Ideas from a brainstorm disappear with the conversation. An agent needs a place to keep the few worth saving, under a topic, and a way to find them later by meaning rather than by the words used that day.

## What Changes

- `c3po idealake` stores short ideas organized by topic. An idea is one line of at most 300 characters. A topic is one line of at most 80 characters.
- An agent adds the chosen texts, for example `c3po idealake add xyz "second idea" "third idea"`. Brainstorm numbers are not ids.
- `list`, `show`, `remove`, and `topics` read that record. `search <topic> <query>` ranks that topic with xAI embeddings, model `v1`, by how close each idea is to the query. `contains <text> [topic]` is the lowercase text match. Search does not fall back to that match.
- `--json` before the command prints JSON and omits the vectors.
- Ideas are rows in the c3po sqlite file. nebula s3 backs that file up the same way other c3po records are backed up.
- There is no idealake screen.

## Impact

- Specs: `openspec/specs/idealake/spec.md`, `openspec/specs/scout/spec.md`
- Code: `idealake/`, `internal/scout`, `internal/usage`
