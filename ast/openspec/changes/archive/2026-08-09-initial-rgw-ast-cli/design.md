## Context

Agents need a uniform boundary for large codebases. Policy must not scatter
`.rgw-ast.toml` files across repos. The home workspace already places personal
CLIs under `Apps/` and uses OpenSpec for behavioral contracts.

## Goals / Non-Goals

**Goals:**

- Ship a working Go CLI that measures LOC, decides enforcement globally, and
  exposes map/show/search plus hash/read/patch.
- One config path only: XDG `config/rgw-ast/config.toml`.
- Default threshold 5000; mode `auto`.
- Testable pure packages; `go test ./...` green.

**Non-Goals:**

- MCP stdio server or agent host installers (later).
- Full AST rewrite engine / ast-grep dependency for v1.
- Per-repo policy files or monorepo package-level overrides.
- Replacing git, formatters, or package managers.

## Decisions

1. **Global config only** — Load `config.toml` from
   `$XDG_CONFIG_HOME/rgw-ast` or `~/.config/rgw-ast`. Never look for project
   config. Missing file → write defaults then load.
2. **Root = git root or cwd** — Walk parents for `.git`; else use absolute cwd.
   Measure that tree with global globs.
3. **LOC definition** — Count physical lines in files matching include globs,
   skipping exclude globs. Binary/null-byte files skipped. Cache optional under
   XDG cache keyed by root path + mtime fingerprint is nice-to-have; v1 may
   remeasure each `status`/`measure` call for correctness.
4. **Enforcement** — `always` → enforced; `never` → not; `auto` → loc ≥
   `threshold_loc`. Commands that mutate or bulk-read check this flag.
5. **Intelligence v1** — Prefer `go/parser` for `.go`. For other languages use
   lightweight line heuristics (function/class/def patterns) and capped content
   slices. Good enough for structure-first without native AST deps.
6. **Patch model** — Exact unique old→new text replacement; require
   `--expect-hash` matching current SHA-256 of file; atomic write via temp +
   rename.
7. **CLI style** — Flat verbs like other Ryan apps (`help`, `version`,
   `status`); flags after subcommand where needed (`--json`, `--lines`,
   `--expect-hash`).

## Risks / Trade-offs

- Heuristic map/show is weaker than tree-sitter; acceptable for v1 token
  control and can improve later.
- Measuring large monorepos is CPU-bound; may need cache soon.
- Without host hooks, agents can still bypass via shell; CLI enforces only when
  agents choose to call it (documented contract).
