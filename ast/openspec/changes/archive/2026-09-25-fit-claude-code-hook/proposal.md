## Why

Claude Code runs one global PreToolUse hook for every session. The hook
judged every call by the session directory and by substring patterns. The
result was false denials: a Write to `/tmp` from an enforced repo, `2>&1`,
a heredoc into `rgw-ast create --stdin`, and `cp repo/file /tmp/x`. Worse,
it gave false allows: a session started in `/tmp` could edit an enforced
repository. Doctor only looked for project-level host settings, so a hook
installed once in `~/.claude/settings.json` read as missing.

## What Changes

- File tools are judged by the file they change. A target outside every git
  worktree, or ignored by its repository, is allowed. Otherwise the enforcement
  of the containing repository decides.
- The hook uses the event `cwd` that Claude Code sends, unless `--root` is
  passed.
- Shell commands are analysed with a quote-aware lexer instead of substring
  patterns. The lexer covers control operators, pipelines, command and
  process substitution, heredocs, redirections, `cd`, and nested
  `sh -c`/`eval`. It judges only the paths a command writes, with the same
  rule as file tools.
- File-descriptor duplication, `/dev/null`, input redirection, and heredocs
  are not mutations. `rgw-ast` invocations may use them and may sit in
  compound commands. A mutation anywhere in the command is still denied.
- The workspace is measured only when a command has a write target, so
  read-only calls stay fast.
- Doctor also accepts the user-level Claude Code settings
  (`$CLAUDE_CONFIG_DIR` or `~/.claude`) and the user-level Codex hooks
  (`$CODEX_HOME` or `~/.codex`) as hook evidence.

## Capabilities

### Modified Capabilities
- `agent-hook`: target-scoped decisions and quote-aware shell analysis
- `doctor-bootstrap`: user-level host hook evidence

## Non-goals

- Tracing writes made by arbitrary interpreters (`python -c`, `node -e`) or
  by scripts run from files
- Evaluating shell variables beyond `$HOME` and `~`. Any other expanded
  target is judged by the current directory.
- Per-project policy files. Policy stays global.
