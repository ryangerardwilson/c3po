## 1. Spec

- [x] 1.1 Add the dd delete requirement
- [x] 1.2 Validate the change with `openspec validate o-dd-delete --strict --no-interactive`

## 2. Screen

- [x] 2.1 Arm on `d` and delete the current file or directory on the second `d`
- [x] 2.2 Cancel the arm with Esc and with any other key
- [x] 2.3 Name `dd` in the help

## 3. Check

- [x] 3.1 Run `GOWORK=off go test ./o/...`
