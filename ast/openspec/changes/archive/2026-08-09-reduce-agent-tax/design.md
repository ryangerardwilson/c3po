## Context

v1 remeasured on every command and only blocked include-matched whole reads.

## Goals / Non-Goals

**Goals:** Sub-second enforced decisions on large trees via cache; close read
holes; lean search; optional `--root`; host hook JSON API.

**Non-Goals:** MCP, complete AST graphs, per-project config.

## Decisions

1. Cache key = sha256(root + include/exclude/threshold) + directory mtime sample
   of root (and optional git HEAD if present). Store loc, file_count, enforced
   inputs in JSON under cache dir.
2. `CountCached` loads cache if fingerprint matches; else counts and writes.
3. Whole-file deny: when enforced && deny_whole_file_read, reject any regular
   file that is not binary (NUL probe), not only PathMatches.
4. Search snippets max 120 runes, ellipsis if truncated.
5. `--root PATH` before or after subcommand; resolves absolute workspace root.
6. Hook reads JSON from stdin: `{ "tool_name": "...", "tool_input": {...} }`
   (Claude/Codex-like). Denies known mutation tools and freeform shell writes
   when workspace is enforced, unless command is clearly `rgw-ast`.

## Risks

- Cache staleness if files change without touching root mtime — mitigate with
  short max age (e.g. 60s) plus root mtime + optional HEAD.
