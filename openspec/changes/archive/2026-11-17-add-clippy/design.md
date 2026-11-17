## Context

Agents already hand the person a draft in the transcript. The person then selects it. `wl-copy` is the Wayland clipboard on this machine. Nebula crypt already pipes a secret to `wl-copy` on stdin and returns while the clipboard server stays open.

## Goals / Non-Goals

- Goals: one command an agent can run with a heredoc or a multiline argument. The clipboard receives the exact bytes. The transcript does not gain another copy of the text.
- Non-goals: reading the clipboard, a history, a screen, X11 `xclip`, or storing the text.

## Decisions

- `c3po clippy` with no arguments reads stdin to EOF. On a terminal, that invocation prints help instead of waiting.
- Arguments are joined with one space. A newline inside an argument stays. `help` and `version` are commands, so copying the word `help` is done through stdin.
- `wl-copy` receives the text on stdin and never as an argument. Stderr is attached only when it is an `*os.File`, so a pipe on stderr cannot keep the process waiting on the forked clipboard server.
- Success stdout is exactly `copied`. The text is absent from stdout and stderr. Empty text exits 2 before `wl-copy` runs. A missing `wl-copy` exits 1 with `wl-copy is required.`
- The usage counter records `clippy` and does not store the text. That is the existing usage rule.

## Risks / Trade-offs

- A clipboard server that holds stderr would block the command. The `*os.File` check is the same guard nebula crypt uses.
- Joining arguments with spaces means two words are one line. Multiline text belongs in stdin or in one argument.

## Migration Plan

- No stored data. Install the new binary. Agents call `c3po clippy`.

## Open Questions

- None.
