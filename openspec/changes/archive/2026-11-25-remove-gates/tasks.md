## 1. Remove

- [x] 1.1 Delete the `gates` package and unregister it from scout, help, and the usage catalog
- [x] 1.2 Treat `c3po gates` as an unknown command in the scout and usage tests
- [x] 1.3 Drop `gates` from the README, `AGENTS.md`, and `openspec/config.yaml`

## 2. Check

- [x] 2.1 Run `GOWORK=off go test ./internal/scout/ ./internal/usage/ -count=1`
- [x] 2.2 Validate with `openspec validate remove-gates --strict --no-interactive` and archive it
