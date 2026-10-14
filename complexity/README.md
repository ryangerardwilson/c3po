# complexity

`complexity` (alias `cx`) is the system-wide complexity gate for this machine. It measures cognitive complexity, nesting, function length, and parameter count for Go, Python, Rust, and TypeScript/JavaScript, plus relationship degree, depth, cycles, and naming for ORM models and SQL. It parses source only. It does not compile the target and it does not need a language runtime.

The implementation is CGO-free Go. Go uses the standard parser. The other languages use a syntactic scanner. That is a deliberate departure from a tree-sitter build so the binary stays static, matching the other Apps CLIs.

## Gate

`complexity check` fails when a function is over a threshold. `complexity check --diff HEAD --ratchet` fails only when new code exceeds a threshold or an existing score gets worse. A legacy hotspot can stay when an edit does not raise its cognitive complexity, nesting, length, or parameter count. Every path cx prints or records is relative to the directory the command was run from.

Default limits are cognitive 15, nesting 4, 80 non-comment lines, 6 parameters, and cyclomatic 10. Tests, vendored trees, and generated files are excluded. `--include-tests` puts tests back. A scan does not enter a nested git checkout.

`complexity check` prints a Hotspots list only for functions that exceed a limit. A clean run prints the summary and does not list functions that are inside the limits. `complexity rank cognitive 20` orders functions by a metric and exits 0. The metrics are cognitive, nesting, lines, params, and cyclomatic. `top` is `rank cognitive`.

## Layout on this machine

Policy is `$XDG_CONFIG_HOME/complexity/settings.json`, or `~/.config/complexity/settings.json` when `XDG_CONFIG_HOME` is unset. A project `complexity.toml` or `.complexity.toml` overrides it up to the git root, and CLI flags win. The exclude list covers the trees this laptop actually produces: `node_modules`, `dist`, `vendor`, `.next`, `.venv`, coverage, serverless, gradle, generated protobuf, and the test globs `*_test.go`, `test_*.py`, `*.test.*`, `*.spec.*`, `tests/`, `__tests__/`, and `test/`.

`Apps/` is one git root. `Work/` checkouts such as American Dream and Wiom are nested git repositories and are measured from inside those roots, not as one combined tree.

## Models

`complexity models` and `complexity schema` score Prisma, SQL `CREATE TABLE`, SQLAlchemy/SQLModel, and GORM. A cycle is a violation. Table names should look like `<model>_<feature>[_<sub_feature>][_logs]`. Columns should run primary key, relationship ids, feature columns, optional `data`, then timestamps. Naming and column order are reported by default and fail the process when `conventions.strict` or `--strict-conventions` is set. A feature segment that does not match a class or module is a warning.

## Agent hook

`complexity hook` is a PreToolUse and Stop command. PreToolUse denies an editor write that introduces a ratchet violation. Stop blocks unless `complexity check --diff HEAD --ratchet` is clean. Events that cannot be judged are allowed. Install the binary before adding the hook to `~/.claude/settings.json`.

```bash
just install complexity
complexity status --json
complexity check --diff HEAD --ratchet --format json --hotspots-only
```
