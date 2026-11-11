## Tasks

- [x] Record one daily count per feature, actor, and UTC day in the sqlx database `c3po`
- [x] Classify the actor as human, agent, or service, and keep arguments out of the record
- [x] Add `c3po usage` and name it in help
- [x] Validate with `openspec validate feature-usage --strict --no-interactive`
- [x] Run `GOWORK=off go test -C /home/ryan/OpenSource/c3po ./internal/usage/ ./internal/scout/ ./internal/sqlxdb/`
