## 1. Command

- [x] 1.1 Print `<path>: <tally>` with the colon against the path
- [x] 1.2 Describe that line in the `c3po dirt` help, `README.md`, `AGENTS.md`, and `openspec/config.yaml`

## 2. Check

- [x] 2.1 Expect `<path>: <tally>` in `dirt/dirt_test.go`
- [x] 2.2 Run `GOWORK=off go test ./dirt/ -count=1`
- [x] 2.3 Validate this change with `openspec validate dirt-colon --strict --no-interactive` and archive it
