## 1. Spec

- [x] 1.1 Let a colon line that starts with `!` run in the shell
- [x] 1.2 Validate the change with `openspec validate o-colon-bang --strict --no-interactive`

## 2. Screen

- [x] 2.1 Run `:!` in the listed directory and refuse an empty command
- [x] 2.2 Keep touch, mkdir, and rm as the colon commands that do not start with `!`
- [x] 2.3 Name `:!` in the shortcut list and the help

## 3. Check

- [x] 3.1 Run `GOWORK=off go test ./o/...`
