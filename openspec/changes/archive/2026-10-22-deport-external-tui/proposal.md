## Why

`c3po deport` will signal whatever holds the port it is given, including system services. The ports a person actually wants to free are the local servers around 8080 and 8082. The command also has no screen for seeing which of those ports are taken.

## What Changes

- **BREAKING** `c3po deport <port>` signals a process only when the port is from 8000 through 8999. Any other port is internal and is refused with no signal.
- `c3po deport list` prints the occupied ports in that band.
- `c3po tui deport` lists those same ports. `j` and `k` move. `dd` frees the selected port.
- The screen joins the `c3po tui` list.

## Capabilities

### New Capabilities

- `deport`: Free only a safe external port, list the occupied ones, and free one from the screen.

### Modified Capabilities

- `scout`: The terminal screens include `deport`.

## Impact

`deport` and the `c3po tui` dispatch. No new dependency. A port outside 8000–8999, including 22, 53, 3000, and 5173, is no longer killed.
