## Why

An evaluation of rgw-ast 0.1.0 from an agent's side (`~/FEATURE_REQESTS.md`,
2026-10-05) found that the boundary creates confusion where its written rules
and its hook disagree, and friction where enforcement buys no safety:

- The managed block says "read only with `rgw-ast read`", but the hook allows
  whole-file host `Read` and `cat`. Shell analysis blocks `sed -i` but allows
  `python -c "open(...,'w')"` and `openspec archive` outside `rgw-ast exec`.
  Agents learn the rules are optional.
- There is no move. `git mv` is denied, yet workspace relocation procedures
  require moves.
- Markdown counts toward the LOC threshold, so documentation-only
  repositories are enforced, and creating a brand-new note in an enforced
  repository is denied for host `Write`.
- `map`, `show`, `search`, and trace walk different trees than `measure`:
  a linked worktree (`.git` file) nested in a repository is indexed, and
  gitignored artifacts and tool caches appear as symbols.
- Several errors are dead ends: ambiguous `impact --json` has no
  `assessment`, a non-unique patch match gives no line numbers, a plain
  `git diff` is rejected by `apply`, absolute paths outside the session root
  fail, `read --strict-lines` does not fail past end of file.
- The managed block is long, jargon-heavy, and duplicated into every nested
  AGENTS.md.

## What Changes

- **Hook**: host `Read` is judged by the same bound as `rgw-ast read`
  (allowed when bounded by `max_read_lines` or the file is short). `Write`
  to a path that does not exist is allowed (it is a create). Inline
  interpreter code (`python -c`, `node -e`, …) with file-writing calls is a
  write. A command that matches `generators.allow` must run through
  `rgw-ast exec` (read-only `openspec` subcommands excepted). Deny reasons
  name the exact replacement command and, for an existing file, its current
  hash.
- **Move**: new `rgw-ast move <src> <dst> --expect-hash <sha>`, and a `move`
  op in `apply` manifests. `git mv` deny reasons point to it.
- **Measure**: files matching the new global `prose` globs (default
  Markdown, reStructuredText, plain text) are reported as `prose_loc` and do
  not count toward `threshold_loc` unless `enforcement.count_prose` is true.
- **One walk scope**: `map`, `show`, `search`, `explain`, and trace use the
  measure scope: nested repositories (a `.git` directory or file), the root
  `.gitignore`, and directories that ignore themselves (`.gitignore` of `*`,
  as tool caches write). `.ruff_cache` and similar caches are skipped by name.
- **Trace**: an ambiguous query always carries
  `assessment.conclusion: "ambiguous"`.
- **Patch**: a non-unique match reports every matching line; `--all`
  replaces every occurrence. Hash mismatches say how to recover.
- **Apply**: accepts a plain unified diff (`git diff`, `diff -u`) without
  `# path:` headers; a diff hunk's context is its guard so `expect_hash` is
  optional for diff edits. `--no-format` skips formatting. Output always
  reports the formatter (`none` when none ran).
- **Read**: header reports the total line count and, when clamped, the next
  range. `--strict-lines` fails when the range passes end of file.
- **Paths**: an absolute path outside the session root resolves its own
  workspace root, as the hook already does.
- **Search**: `--offset`, `--regex`, `-i`, and a total match count.
- **CLI**: `--version`/`-v`; `show <path>` falls through to `map <path>`;
  `explain <dir>` reports visible child counts.
- **Doctor/block**: a shorter managed block. A workspace without its own
  block is `inherited` (ready) when an ancestor AGENTS.md has the current
  block.

## Capabilities

### New Capabilities
- `file-move`: hash-guarded move or rename

### Modified Capabilities
- `agent-hook`: bounded host reads, create-by-Write, interpreter writes,
  generator wrapping, actionable deny reasons
- `measure-enforcement`: prose excluded from the threshold
- `global-config`: `prose` globs and `enforcement.count_prose`
- `file-boundary`: read header and strict bound, patch match reporting and
  `--all`, plain-diff apply
- `code-intelligence`: one walk scope, search paging/regex, show/explain
  fallbacks, ambiguous trace assessment
- `cli-surface`: `move` verb, `--version`, out-of-root absolute paths
- `doctor-bootstrap`: inherited agents block, shorter canonical block

## Non-goals

- Tracing writes by interpreters running script files (`python x.py`) or by
  arbitrary binaries. Inline code is scanned for known write calls only.
- Policing shell reads (`cat`, `grep`); only the host `Read` tool is bounded.
- A host-side rewrite that turns `Edit` into `patch` inside the hook.
- `--expect-hash last` or any session state between invocations.
- Per-project policy files. Policy stays global.

## Impact

- Workspaces whose LOC was mostly Markdown may drop below the threshold and
  stop being enforced. `enforcement.count_prose = true` restores the old
  count.
- Existing AGENTS.md blocks become `stale` until refreshed from
  `rgw-ast agents-block`; nested copies can be removed and become
  `inherited`.
- No migration or credential impact. Rollback: reinstall the previous binary.
