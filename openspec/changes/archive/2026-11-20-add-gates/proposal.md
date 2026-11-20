## Why

An agent judging its own work needs a fixed statute to judge against. When the rules live in a prompt or a file the agent can reach, the agent can soften them. The text belongs in the c3po binary, where only a changed and reinstalled c3po can change it.

## What Changes

- `c3po gates` prints hardcoded legislation by type. The types are `code` and `essays`, in that order.
- A gate has a section number, a title, and a body. Sections order numerically, component by component, so `2` comes before `10` and `1` before `1.1`.
- The only published gate is code section `1`, `Architecture guardrails`, embedded from `gates/statute/code/1.md` with `go:embed`.
- `list` and `show` print a type's gates with the sequential line: a failed gate fails every later gate, and those are not judged. `show <type> <section>` prints one gate.
- An unknown type, an unknown section, and any other subcommand, including `edit`, exit 2. There is no config file, user directory, or add, edit, or remove command.
- c3po does not scan, call a model, or decide pass or fail. The agent judges and writes the report.
- There is no gates screen.

## Capabilities

### New Capabilities

- `gates`: print the hardcoded gates an agent judges work against.

### Modified Capabilities

- `scout`: `gates` is an app in the command list, the help list, and the usage catalog.

## Impact

- Specs: `openspec/specs/gates/spec.md`, `openspec/specs/scout/spec.md`
- Code: `gates/`, `internal/scout`, `internal/usage`
- Docs: `AGENTS.md`, `README.md`, `openspec/config.yaml`
- Migration: none. The source version stays 0.5.0.
