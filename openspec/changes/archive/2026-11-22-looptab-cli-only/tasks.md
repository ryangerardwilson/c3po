## 1. CLI only

- [x] 1.1 Make no arguments print the help and remove `edit` and the vim launcher
- [x] 1.2 Export `JobsFile` and `Install` on the existing install function
- [x] 1.3 Remove the looptab screen code and its registration in `screen`
- [x] 1.4 Export the session helpers in `screen` for a screen r2d2 owns

## 2. Check

- [x] 2.1 Test no arguments, `edit`, the help, the template, and install
- [x] 2.2 Test that `screen` does not open looptab
- [x] 2.3 Run `GOWORK=off go test ./looptab/ ./screen/`
- [x] 2.4 Validate with `openspec validate looptab-cli-only --strict --no-interactive` and archive it
