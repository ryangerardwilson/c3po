## 1. Path

- [x] 1.1 Put the scheduler file and the template under `<config>/c3po/looptab`, honoring an absolute `XDG_CONFIG_HOME`
- [x] 1.2 Refuse to register a path that opens the scheduler file
- [x] 1.3 Name the new scheduler path in `AGENTS.md`

## 2. Check

- [x] 2.1 Test the path with an absolute, a relative, and an empty `XDG_CONFIG_HOME`
- [x] 2.2 Test that a symlink to the scheduler file is refused
- [x] 2.3 Run `GOWORK=off go test ./looptab/`
- [x] 2.4 Validate with `openspec validate looptab-config-under-c3po --strict --no-interactive` and archive it
