## 1. Command

- [x] 1.1 Classify each porcelain line and print `<path> : <tally>` per dirty repository, with a path under home written as `~/...`
- [x] 1.2 Describe the tally line in the `c3po dirt` help
- [x] 1.3 Describe the tally line in `README.md`, `AGENTS.md`, and `openspec/config.yaml`

## 2. Check

- [x] 2.1 Update the temporary-repository tests in `dirt/dirt_test.go` to the tally line, including added, deleted, modified, and untracked marks
- [x] 2.2 Run `GOWORK=off go test ./dirt/ -count=1`
- [x] 2.3 Validate this change with `openspec validate dirt-tally --strict --no-interactive` and archive it
