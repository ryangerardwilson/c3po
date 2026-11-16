## 1. Probe first

- [x] 1.1 Watch one agent build `/tmp/cc-probe/app` in Go, TypeScript, and Python through a subagent
- [x] 1.2 Record whether each language landed a function at or above max_cc, and which tool wrote it

Probe, 2026-11-16. One subagent. Eight tool calls. The hook allowed every call. Host Write created `app/roster.go`, `app/roster.ts`, and `app/roster.py`. Gap scored Go `Assign` and both `assign` functions at 35. The subagent then ran `c3po ast delete` on `app/roster.go`. TypeScript and Python at 35 remain under `/tmp/cc-probe/app`. Head stayed `15cb3d8`.

## 2. Gates, after the probe

- [x] 2.1 Recognize MCP tool names in the hook and test a name that contains `mcp__`
- [x] 2.2 Run the max_cc gate on append and on a new Go, TypeScript, and Python file, including a host Write of a path that does not exist yet
- [x] 2.3 Refuse a Go write that does not parse when that would skip the gate
- [x] 2.4 Remove `push_release_upgrade.sh` from `generators.allow`

## 3. Docs, after the probe

- [x] 3.1 Pick one writer for the American Dream Enterprises `pr/*` trees
- [x] 3.2 Regenerate one agents block and drop stale `c3po tui` lines

Second probe, 2026-11-16, after the gate was installed. One subagent. Host Write of `app/roster.go`, `app/roster.ts`, and `app/roster.py` was denied. Each reason named the function at 35 crossing 30. No file was written. Gap over 0. The subagent did not switch to a shell write.
