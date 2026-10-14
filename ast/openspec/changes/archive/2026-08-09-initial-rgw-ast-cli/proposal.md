## Why

Coding agents waste tokens and risk unbounded edits on large repositories by
opening whole files and writing freely. A personal Go CLI (`rgw-ast`) should
provide structure-first inspection and hash-guarded mutation, with automatic
enforcement when a workspace exceeds a global line threshold—without requiring
any per-project config file.

## What Changes

- Introduce the `rgw-ast` Go CLI under `Apps/rgw-ast` with OpenSpec as the
  project truth layer.
- Single global policy at `~/.config/rgw-ast/config.toml` (XDG-aware); create
  with defaults on first use.
- Commands: `help`, `version`, `config`, `status`, `measure`, `map`, `show`,
  `search`, `hash`, `read`, `patch`, and local `install` path via `install.sh`.
- Discover workspace root (git root, else cwd); measure LOC with global
  include/exclude; set `enforced` when `mode=auto` and loc ≥ threshold
  (default 5000), or when `mode=always`.
- When enforced: reject whole-file reads of source files; require fresh
  SHA-256 before exact-text patches.
- Structure-first tools: file/symbol map, show symbol body, capped search.

## Capabilities

### New Capabilities

- `cli-surface`: top-level commands, help, exit codes, output shape
- `global-config`: sole policy file location, defaults, ensure-on-first-use
- `measure-enforcement`: root discovery, LOC measure, threshold, enforced flag
- `code-intelligence`: map, show, search contracts
- `file-boundary`: hash, bounded read, hash-guarded exact patch
- `distribution`: version stamp, install.sh local install path

### Modified Capabilities

- (none — greenfield project)

## Impact

- New app tree: `Apps/rgw-ast/**`
- Runtime config: `~/.config/rgw-ast/config.toml`
- Optional cache: `~/.cache/rgw-ast/`
- Install target: `~/.local/bin/rgw-ast` via `./install.sh from .`
- Non-goals: MCP server, Codex/Claude hook installers, callers/deps graphs,
  tree-sitter multi-language rewrite engine, per-project config overrides
