## 1. Command

- [x] 1.1 Replace the embedded essay with a table of architecture provisions, one bullet or short paragraph per body, at most 300 code points each
- [x] 1.2 Add the statute level to `list`, add `get <id>`, and remove `show`
- [x] 1.3 Refuse an unknown type, statute, or id, `edit`, `add`, `remove`, and extra words with exit 2 and nothing on stdout
- [x] 1.4 Move the sequential rule to the help and give the `<id>  pass` report line
- [x] 1.5 Update the c3po usage line, `README.md`, and `AGENTS.md`

## 2. Check

- [x] 2.1 Test the list levels, the provision lines, `get` printing only the body, the top section bodies, and subsection numbering without gaps
- [x] 2.2 Test that every body is at most 300 code points and that the kept phrases still occur
- [x] 2.3 Test the section order `S.1`, `S.2`, `S.2.3`, `S.3`, `S.3.5.1`, `S.10` and that `S.1.0`, `S.01`, and `1` are refused
- [x] 2.4 Run `GOWORK=off go test ./gates/ ./internal/scout/ ./internal/usage/` and `c3po ast gap`
- [x] 2.5 Validate with `openspec validate gates-provisions --strict --no-interactive` and archive it
