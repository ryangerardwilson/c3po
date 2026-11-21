## 1. Command

- [x] 1.1 Resolve a section argument written as `§1.1`, `1.1`, or `S.1.1` to the published `§1.1`
- [x] 1.2 Print several sections in the order written, each as the canonical section, its body, and a blank line between
- [x] 1.3 Keep one section to only its body and one newline
- [x] 1.4 Refuse the whole pass with exit 2 and empty stdout when any section is unknown
- [x] 1.5 Show the three section forms and several sections in the help and the usage line
- [x] 1.6 Update `README.md`, `AGENTS.md`, and the gates sentence in `/home/ryan/AGENTS.md`

## 2. Check

- [x] 2.1 Test that `§6`, `6`, and `S.6`, and `§8.3`, `8.3`, and `S.8.3`, print the same body
- [x] 2.2 Test that `6 7 8.3` prints the labelled bodies in the order written with no trailing blank line
- [x] 2.3 Test that `6.0`, `01`, `0`, `S.0`, and `s.1` are unknown sections and that `6 9` prints nothing and names `9`
- [x] 2.4 Test that the help shows `§1.1`, `1.1`, `S.1.1`, and several sections
- [x] 2.5 Run `GOWORK=off go test ./gates/ ./internal/scout/ ./internal/usage/` and `c3po ast gap`
- [x] 2.6 Validate with `openspec validate gates-section-forms --strict --no-interactive` and archive it
