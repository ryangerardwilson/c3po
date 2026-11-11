## 1. Contract
- [x] 1.1 Write proposal and spec deltas
- [x] 1.2 Validate strictly

## 2. Gate
- [x] 2.1 `ApplyOpts.Check` sees the formatted bytes of every edit and create before any write, dry run included
- [x] 2.2 `CreateAbsentCheck` checks content before any directory or file is made
- [x] 2.3 Symbol patch, create, and apply refuse with the plain patch rule, sorted by path then name, with no `Error:` line
- [x] 2.4 Tests for symbol patch, create, and apply: crossing, new high, already-high rise, removal, non-Go, unparsed, dry run

## 3. Gap
- [x] 3.1 Export the one-file scorer as `cyclotree/score` and use it in cyclotree
- [x] 3.2 `c3po ast gap` over `measure.Scope.Walk` with text, `--json`, and `--check`
- [x] 3.3 Tests for Go at and under the line, Python, skipped vendor and gitignored files, JSON, check, empty debt, `--root`, and usage

## 4. Docs
- [x] 4.1 `ast gap` in the usage catalog after `ast explain`
- [x] 4.2 `c3po ast help`, `help patch`, `help apply`, `help create`, `help gap`, README, and the agents block
