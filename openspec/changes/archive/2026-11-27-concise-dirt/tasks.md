## 1. Command

- [x] 1.1 Count the non-`##` porcelain lines that pass the inside filter and print `<path>  <count>` per dirty repository with no blank lines
- [x] 1.2 Describe the one-line list in the `c3po dirt` help
- [x] 1.3 Describe the one-line list in `README.md`, `AGENTS.md`, and `openspec/config.yaml`

## 2. Check

- [x] 2.1 Update the temporary-repository tests in `dirt/dirt_test.go` to the one-line format, including a rename, an untracked directory, a scan inside a repository, and a failed status
- [x] 2.2 Run `GOWORK=off go test ./dirt/ -count=1`
- [x] 2.3 Validate this change with `openspec validate concise-dirt --strict --no-interactive` and archive it
