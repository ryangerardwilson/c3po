## 1. Spec

- [x] 1.1 Add the cut, copy, and paste requirement
- [x] 1.2 Validate the change with `openspec validate o-cut-copy-paste --strict --no-interactive`

## 2. Screen

- [x] 2.1 Stage the current file or directory with `x` and `y`
- [x] 2.2 Paste it into the listed directory with `p`
- [x] 2.3 Refuse a name collision and a directory pasted into itself
- [x] 2.4 Name `x`, `y`, and `p` in the help

## 3. Check

- [x] 3.1 Run `GOWORK=off go test ./o`
