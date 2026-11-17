## Why

An agent can write a multiline draft, but the person still has to copy it out of the transcript by hand. The draft should land on the clipboard in one command, without being printed again.

## What Changes

- `c3po clippy` copies a multiline text to the Wayland clipboard with `wl-copy`.
- With no arguments, the text is stdin. Arguments are one string joined by spaces, and a newline inside an argument is kept.
- The text is not printed. Success prints `copied`. An empty text is refused.
- There is no clippy screen.

## Capabilities

### New Capabilities

- `clippy`: copy a multiline text to the clipboard and do not print it.

### Modified Capabilities

- `scout`: `clippy` is an app in the command list and the help list.

## Impact

- Specs: `openspec/specs/clippy/spec.md`, `openspec/specs/scout/spec.md`
- Code: `clippy/`, `internal/scout`, `internal/usage`
- The workstation needs `wl-copy`. The text is not stored.
