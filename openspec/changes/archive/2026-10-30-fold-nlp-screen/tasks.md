## 1. Spec
- [x] 1.1 Add the nlp spec and the scout and tui deltas

## 2. Screen
- [x] 2.1 Move the trainer into `nlp/` and register `c3po tui nlp`
- [x] 2.2 Keep courses in the sqlx database `nlp`

## 3. Check
- [x] 3.1 `GOWORK=off go test ./nlp/ ./internal/scout/` passes
- [x] 3.2 `openspec validate fold-nlp-screen --strict --no-interactive` passes
