## 1. Spec

- [x] 1.1 Add the idealake delta for a topic with no ideas
- [x] 1.2 Validate with `openspec validate add-empty-topic --strict --no-interactive`

## 2. Command

- [x] 2.1 Store a topic when add is given no ideas
- [x] 2.2 List that topic with a count of 0
- [x] 2.3 Keep idea adds, stdin ideas, and rejected ideas unchanged

## 3. Check

- [x] 3.1 `GOWORK=off go test` for idealake
- [x] 3.2 Archive the change and validate the idealake spec
