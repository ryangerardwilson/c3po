## Context

`c3po ast gap` already scores Python. The write gate in `ccgate.go` scores Go only. `cmdAppend` stores bytes after the hash check and does not call `ccCheck`. The hook allows a host Write when the path is absent. `normalizeTool` removes `_` and then looks for `__`.

## Goals / Non-Goals

- Goals: record the gates that must change after the agent probe. Enforce one max_cc line for Go, TypeScript, and Python.
- Non-Goals: implementing those gates in this document. The probe has run. Product code stays unchanged until the gate work starts.

## Decisions

- The probe is a scratch repo at `/tmp/cc-probe`. It is not a product change.
- A function at or above max_cc is refused on create, patch, apply, and append for `.go`, `.ts`, `.tsx`, `.py`. The existing Go exception stays: a function already at or above the line may rise.
- MCP names are matched after the server prefix is removed, not after every `_` is deleted.

## Risks / Trade-offs

- TypeScript and Python scoring may disagree with a later formatter.

## Migration Plan

- No stored data moves. Docs are a separate follow-up.

## Probe

On 2026-11-16 one Claude subagent built `/tmp/cc-probe/app` in a repo the live hook treated as enforced (loc 5784 after the writes, threshold 5000, head 15cb3d8). The parent wrote nothing. Every `c3po ast hook` decision was allow. Eight tool calls.

The subagent used the host Write tool for three new files. Each file was one function with 32 separate ifs. `c3po ast gap` scored each at 35.

- `app/roster.go` function `Assign` was written. The hook did not refuse that new-file Write. `c3po ast create` would have refused it. After gap printed `35 app/roster.go:Assign go`, the subagent deleted the file with `c3po ast delete`.
- `app/roster.ts` function `assign` remains at 35.
- `app/roster.py` function `assign` remains at 35.

No shell command wrote file contents. The only c3po filesystem change was the Go delete. `pad.go` and head `15cb3d8` stayed.

## Open Questions

- Closed by the probe. A host Write of a new `.go`, `.ts`, `.tsx`, or `.py` file must be gated, or denied in favor of `c3po ast create`. Gating only `c3po ast create` does not bind the path the agent used.
