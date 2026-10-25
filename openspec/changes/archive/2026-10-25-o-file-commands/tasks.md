## 1. Spec

- [x] 1.1 Limit the colon line to mv, rm, cp, touch, and mkdir, and drop its hint
- [x] 1.2 Validate the change with `openspec validate o-file-commands --strict --no-interactive`

## 2. Screen

- [x] 2.1 Run those commands, including `rm -rf`, and refuse anything else
- [x] 2.2 Show a colon line with no hint
- [x] 2.3 Name the commands in the shortcut list and the help

## 3. Check

- [x] 3.1 Run `GOWORK=off go test ./o/...`
