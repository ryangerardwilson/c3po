## 1. Spec

- [x] 1.1 Add the full-height and command requirements
- [x] 1.2 Validate the change with `openspec validate o-full-screen --strict --no-interactive`

## 2. Screen

- [x] 2.1 Paint one line per terminal row, with the key hint on the last line
- [x] 2.2 Run touch, mkdir, and rm from `!` in the listed directory
- [x] 2.3 Refuse any other command or flag, and cancel on Esc
- [x] 2.4 Name `!` in the help

## 3. Check

- [x] 3.1 Run `GOWORK=off go test ./o`
