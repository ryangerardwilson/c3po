# rgw-ast

Global AST-aware agent boundary CLI. One policy file for every repository:

```text
~/.config/c3po/ast/config.toml
```

(or `$XDG_CONFIG_HOME/c3po/ast/config.toml`)

No per-project config is required. When `enforcement.mode = "auto"` (default)
and measured source LOC is at least `threshold_loc` (default **5000**),
`rgw-ast status` reports `enforced: true` and whole-file reads / unhashed
patches are rejected for included source files.

## Spec-driven development

This repo uses [OpenSpec](https://github.com/Fission-AI/OpenSpec). Accepted
capabilities live under `openspec/specs/`.

## Commands

```bash
rgw-ast help
rgw-ast version [--json]
rgw-ast config

rgw-ast [--root <path>] status [--json] [--refresh]
rgw-ast [--root <path>] measure [--json] [--refresh]
rgw-ast [--root <path>] doctor [--json] [--refresh]
rgw-ast agents-block

rgw-ast [--root <path>] map [path]
rgw-ast [--root <path>] show <symbol|path:symbol|path>
rgw-ast [--root <path>] search [--path <dir>] [--glob <pat>] [--offset N] [--regex] [-i] <query>
rgw-ast search --help
rgw-ast [--root <path>] explain <path>
rgw-ast [--root <path>] callers [--json] [--limit N] [--offset N] <symbol>
rgw-ast [--root <path>] callees [--json] [--limit N] [--offset N] <symbol>
rgw-ast [--root <path>] impact [--json] [--limit N] [--offset N] <symbol>

rgw-ast [--root <path>] hash <file> [...]
rgw-ast [--root <path>] read <file> --lines START-END [--number] [--strict-lines]
rgw-ast [--root <path>] create <file> --expect-absent (--from-file f|--stdin) [--parents]
rgw-ast [--root <path>] append <file> --expect-hash <sha> (--from-file f|--stdin)
rgw-ast [--root <path>] patch <file> --expect-hash <sha> [--symbol Name | --all] \
  (--old/--new | --old-file/--new-file | --ops-file ops.json)
rgw-ast [--root <path>] apply --from-file <manifest.json|diff> [--dry-run] [--no-format]
rgw-ast [--root <path>] delete <file> --expect-hash <sha> [--prune-empty]
rgw-ast [--root <path>] move <src> <dst> --expect-hash <sha> [--parents]

rgw-ast [--root <path>] exec [--json] -- <generator command...>
rgw-ast [--root <path>] hook
rgw-ast [--root <path>] packages [--json]
rgw-ast [--root <path>] affected [--json] [path...]
rgw-ast [--root <path>] graph [--json]
rgw-ast [--root <path>] neighbors [--json] <id>
rgw-ast [--root <path>] cut [--json] [path]
```

### Behavior notes

- Measure honors `.gitignore`, stops at nested git repos and linked worktrees
  (a `.git` file), skips directories that ignore themselves (a `.gitignore` of
  `*`, as tool caches write), and fingerprints dirty working trees
  (`git status --porcelain`) so nested edits invalidate cache. `map`, `show`,
  `search`, `explain`, and trace walk the same scope.
- Prose (`prose` globs: Markdown, reStructuredText, text) is reported as
  `prose_loc` and does not count toward `threshold_loc` unless
  `enforcement.count_prose = true`, so documentation-only repositories are not
  enforced.
- An absolute path outside the current root uses its own repository's root,
  as the hook does.
- Mutations and the PreToolUse hook recompute enforcement with a fresh measure.
- Delete removes one verified regular file only. It rejects stale hashes,
  directories, symlinks, special files, and outside-root paths; `--prune-empty`
  removes only empty ancestors and never the workspace root.
- When enforced, whole-file `read` is denied for any non-binary text; the
  error names the line count and the first range to read.
- The read header is `# <file> lines A-B of TOTAL`. Oversized `--lines` ranges
  clamp to `max_read_lines`, and the header names the next range
  (`--strict-lines` fails instead, including past end of file). Prefer
  `--number` for citations.
- Search pages with `--offset`; stderr reports `hits A-B of TOTAL; next:
  --offset B`. `--regex` and `-i` are supported. Snippets truncate to 120
  characters; use `--path` / `--glob` to scope.
- A non-unique `patch` match fails and lists every matching line; `--all`
  replaces every occurrence. A stale hash says how to recover.
- `move` renames one hash-verified file (with `git mv` when tracked).
- `map` lists symbols in path order, then line order. `--json` reports
  `truncated` and `next_offset`. `--depth 0` lists only the selected directory.
  It is not ranked by graph degree.
- `callers`, `callees`, and `impact` trace functions. The symbol on an edge is
  the enclosing function, not a nearby const. `impact` JSON includes
  `assessment.conclusion` of `complete` or `partial` (`ambiguous` when several
  symbols match), plus upstream and downstream paths. `partial` is not the
  whole outcome. `graph`, `neighbors`, and `cut` are package structure, not
  that trace.
- `graph --json` returns node and edge arrays. `degree` matches the neighbor
  list. A Go module and a Python package are not given an invented dependency.
- `patch --symbol` formats one function. `apply --from-file` edits, creates,
  deletes, and moves as one hash-checked transaction. It takes a JSON
  manifest, a `# path:` bundle, or a plain `git diff` / `diff -u` (a diff
  edit's context lines are its guard, so the hash is optional). `--dry-run`
  writes nothing; `--no-format` skips formatting; output names the formatter
  or `none`.
- `doctor` `enforcement_status: degraded` means the hook or the agents block
  is not ready. A correct trace does not clear it.
- `version` on a dirty build includes `-dirty` and a source hash.
- Trusted generators (OpenSpec, etc.) must run as `rgw-ast exec -- ...`; global
  `generators.allow` entries are exact token prefixes matched from argv[0].
- The hook judges each write by its target, not the session directory. A
  target outside every git worktree, or ignored by its worktree, is free;
  otherwise that worktree's enforcement decides. Shell text is lexed with
  quotes, heredocs, substitutions, `cd`, and `sh -c` in view, so `2>&1`,
  `/dev/null`, stdin heredocs, and read-only pipelines around rgw-ast pass,
  while any write into an enforced tree is denied. In an enforced tree a host
  `Write` of a missing file is allowed (it is a create); a host `Read` needs a
  `limit` of at most `max_read_lines` unless the file is that short; inline
  interpreter code (`python -c`, `node -e`, `perl -e`, …) with file-writing
  calls is a write; a command matching `generators.allow` must run through
  `rgw-ast exec` (`openspec` only for `new|archive|init|update`). Deny reasons
  name the exact command, with the file's current hash.
- Doctor reads hook evidence from workspace host settings and from the
  user-level `$CLAUDE_CONFIG_DIR`/`~/.claude/settings.json` and
  `$CODEX_HOME`/`~/.codex/hooks.json`.
- Generator reports separate pre-existing dirty paths from content deltas and
  compare ignored-path content before and after execution.
- Doctor compares the embedded AGENTS block with canonical `agents-block`
  output and recognizes hooks only from parsed command fields. A workspace
  with no block of its own is `inherited` when the nearest ancestor AGENTS.md
  with a block has the current one.

## Local verification

```bash
go test ./...
go run ./cmd/rgw-ast help
go run ./cmd/rgw-ast version --json
go run ./cmd/rgw-ast doctor --json
```

## Install

rgw-ast is an app in the home monorepo (`~/Apps/rgw-ast`). It is installed
from source; there are no GitHub releases. After a change, install it so every
agent's `rgw-ast` (and the PreToolUse hook) runs the new build:

```bash
just install rgw-ast        # from ~; same as Apps/rgw-ast/install.sh from Apps/rgw-ast
```

The binary goes to `~/.local/bin/rgw-ast` (or `$RGW_AST_INSTALL_DIR`) and is
stamped with the commit, `-dirty` when this app's tree has uncommitted changes:

```bash
rgw-ast version --json
```
