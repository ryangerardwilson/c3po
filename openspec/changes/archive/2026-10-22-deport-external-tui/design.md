## Context

Deport reads `/proc/net/{tcp,tcp6,udp,udp6}` and the process fd table, then sends SIGTERM and, if the port stays taken, SIGKILL. The port argument is any number from 1 to 65535. Other terminal screens live under `c3po tui`. The Drive screen uses `dd` as an armed chord: the first `d` arms, Esc cancels, and any other key clears the arm and runs as a motion.

## Goals / Non-Goals

**Goals:**

- Signal only ports 8000 through 8999, the band that contains 8080 and 8082.
- List the occupied ports in that band from the command and from a screen.
- Free the selected screen row with `dd`, using the same signal sequence as the command.

**Non-Goals:**

- Killing privileged ports, databases, or dev servers outside that band.
- A second confirm modal beyond the `dd` chord.
- Publishing a new version tag.

## Decisions

The safe band is 8000 through 8999 inclusive. That is the alternate-HTTP range around the ports named in the request. A wider rule such as "anything above 1024" would still include DNS resolvers on high ports, print services, databases, and the local cloudflared port. A short allowlist of only 8080 and 8082 would miss the rest of that same kind of port. The command refuses anything outside the band before it reads a socket table or sends a signal.

The screen is `c3po tui deport`, matching the other screens. `c3po deport list` prints the same rows for a pipe: `<port> <pid> <name>`, sorted by port then pid. A taken port whose process cannot be seen is the port number alone. `dd` calls the existing free path, so the timeout and the second signal stay in one place.

The list is one read of the four socket tables and one walk of the proc tree. Internal ports are dropped while the tables are read, so they are not rows and cannot be selected.

## Risks / Trade-offs

- [A wanted dev server on 3000 or 5173 is out of band] → The help names 8000 through 8999. Widening the band is a later change.
- [An occupied port can appear or vanish between paints] → The screen reloads at the start of each key. The cursor index is kept and clamped.
- [The integration test used to bind an ephemeral port] → It now binds a free port inside the band, so it does not depend on a live occupant of 8082.

## Migration Plan

Install the new binary. Rollback is the previous binary. No config file changes.

## Open Questions

None.
