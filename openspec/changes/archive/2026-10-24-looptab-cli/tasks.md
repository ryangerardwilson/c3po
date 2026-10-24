## 1. Spec

- [x] 1.1 Add the looptab, scout, and gmail deltas
- [x] 1.2 Validate the change with `openspec validate looptab-cli --strict --no-interactive`

## 2. Command

- [x] 2.1 Register, list, show, remove, and edit jobs files in vim, and install them onto the scheduler
- [x] 2.2 Store the catalog in the sqlx database c3po
- [x] 2.3 Keep `c3po tui looptab` as the run-now screen

## 3. Gmail

- [x] 3.1 Add `c3po gmail tidy` and classify unread mail with the Decisions API
- [x] 3.2 Read the key from OPENAI_API_KEY or the Wiom vault item WIOM_OPENAI_API_KEY

## 4. Check

- [x] 4.1 Run `GOWORK=off go test` for looptab, gmail, and scout
- [x] 4.2 Archive the change and validate the looptab and gmail specs
