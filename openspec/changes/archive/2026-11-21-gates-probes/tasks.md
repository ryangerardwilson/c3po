## 1. Command

- [x] 1.1 Replace `list` and `get` with one probe that goes one level deeper per argument: types, statutes, provisions, one body
- [x] 1.2 Print each provision as `<section>  <heading>` and look a body up by its bare section within the named statute
- [x] 1.3 Refuse `list`, `get`, `show`, `edit`, `add`, and `remove` as unknown commands that say the text changes only when c3po is changed and reinstalled
- [x] 1.4 Refuse an unknown type, statute, or section and a fourth argument with exit 2 and nothing on stdout
- [x] 1.5 Rewrite the help around the four probes and the `S.1  pass` report line
- [x] 1.6 Update the c3po usage line, `README.md`, `AGENTS.md`, and `/home/ryan/AGENTS.md`

## 2. Check

- [x] 2.1 Test the four probes, the provision lines, and that the body probe prints only the body
- [x] 2.2 Test that `C/A/S.1`, `S.9`, `S.1.0`, `S.01`, and `1` are unknown sections and that `list` and `get` exit 2
- [x] 2.3 Keep the 300-code-point test and the kept phrase tests
- [x] 2.4 Test that `c3po gates code architecture S.1` is recorded as `gates`
- [x] 2.5 Run `GOWORK=off go test ./gates/ ./internal/scout/ ./internal/usage/` and `c3po ast gap`
- [x] 2.6 Validate with `openspec validate gates-probes --strict --no-interactive` and archive it
