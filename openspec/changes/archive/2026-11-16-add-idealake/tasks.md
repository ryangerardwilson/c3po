## 1. Spec

- [x] 1.1 Add the idealake spec and the scout delta
- [x] 1.2 Validate with `openspec validate add-idealake --strict --no-interactive`

## 2. Command

- [x] 2.1 Add, list, show, remove, topics, and search ideas by topic
- [x] 2.2 Reject an idea over 300 characters or more than one line, and store none of that call
- [x] 2.3 Store rows in the c3po sqlite file and embed them with xAI model v1
- [x] 2.4 Print JSON when `--json` is the first argument, without vectors

## 3. Check

- [x] 3.1 `GOWORK=off go test` for idealake, scout, and usage
- [x] 3.2 Archive the change and validate the idealake spec
