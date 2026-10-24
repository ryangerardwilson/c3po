## Why

Shortcut legends sit on the bottom of the navigator and the other listing screens. The listing loses a row, and the keys stay in view. Those keys should show only while `?` is open, on every c3po screen.

## What Changes

- The navigator drops the key line. The status line is the last line, and the listing uses the freed row. `?` still toggles the shortcut list.
- Deport, Drive browse, the Drive preset list, pass, and the tunnel route list drop the key line and show those keys only while `?` is open. Esc closes that list. `q` still quits.
- The spreadsheet and cyclotree already keep shortcuts behind `?`. The statistics line, the call chain, and the focused path stay.
- `c3po tui l` stays a printer with no shortcut line.
- A command prompt, a filter prompt, the tunnel add form, and a removal confirmation stay.

## Capabilities

### New Capabilities

- `tui`: shortcut text appears only while `?` is open, on every interactive screen.

### Modified Capabilities

- `o`: the screen still fills the terminal, and the last line is the status line.

## Impact

- Specs: `openspec/specs/o/spec.md`, `openspec/specs/tui/spec.md`
- Code: `o/internal/app`, `deport/tui.go`, `gdrive/internal/tui`, `pass/internal/tui`, `tunnel/internal/tui`
