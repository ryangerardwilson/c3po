## 1. Spec

- [x] 1.1 Add the o path spec
- [x] 1.2 Validate the change with `openspec validate o-direct-path --strict --no-interactive`

## 2. Command

- [x] 2.1 Take `c3po tui o <path>` as the path form
- [x] 2.2 Show that form in help and drop the open command
- [x] 2.3 Cover a directory, a missing path, and the word open

## 3. Check

- [x] 3.1 Run `GOWORK=off go test ./o`
