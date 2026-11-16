## Context

c3po apps keep durable rows in the sqlite file `c3po`. That file lives on the runtime tmpfs. A write publishes it with nebula `s3` to `s3://backups/c3po.sqlite`. nebula `sql` is not used. Agents already reach every app through `c3po <name>`.

xAI embeddings are `POST https://api.x.ai/v1/embeddings` with model `v1`. Retrieval uses the prefix `passage: ` on a stored idea and `query: ` on a search. The key is `XAI_API_KEY`.

## Goals / Non-Goals

- Store the ideas a person chooses from a brainstorm, under the topic they name.
- Keep each idea to one line of at most 300 characters so the stored line is the whole idea.
- Let an agent list, show, remove, and search without a screen.
- Rank one topic by cosine similarity of `v1` embeddings. Do not mix in text matches.
- `contains` is a separate lowercase substring search.
- Leave the TUI list unchanged. Idealake is not a screen.

## Decisions

- One idea is one `record` row: app `idealake`, kind `idea`, key the decimal id. The body is JSON with the topic, the text, the time, the model, and the vector. Ids start at 1 and are not reused.
- A call that adds several ideas stores none of them when any idea is empty, longer than 300 characters, or more than one line. The same rule applies to the topic, with a limit of 80 characters. The count is Unicode code points after trimming.
- Arguments are the ideas. When there are no idea arguments and stdin is not a terminal, each non-empty line is an idea. A terminal with no idea arguments exits 2.
- The stored vector is the embedding of `passage: ` plus the idea. Search embeds `query: ` plus the query. Vectors from any other model are ignored. The printed JSON never includes a vector. It includes `embedded`.
- `search <topic> <query>` prints at most 20 ideas in that topic, highest cosine first. An idea can rank first when its text does not contain the query. A word in the text does not put an idea into the ranking. When the query cannot be embedded, the command exits 1 and prints no ideas. `contains <text> [topic]` prints every idea whose text contains that text, ignoring case, and does not call the embeddings API.
- Add stores the idea when the embedding cannot be made, and stderr says it was stored without an embedding. The key is never printed.
- `--json` is the first argument. Text output is for a person. JSON is for an agent.
- `search` requires a topic. `list` and `contains` take an optional topic. The topic match is exact after trimming.

## Risks / Trade-offs

- Ranking loads every idea in the topic and compares vectors in process. That fits a personal lake. It is not an index.
- A failed s3 publish does not undo the sqlite write. That is the existing record path.
- Ideas without a vector are skipped by search. `contains` still finds their words.
