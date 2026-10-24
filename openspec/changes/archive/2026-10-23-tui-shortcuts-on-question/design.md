## Context

The navigator paints a key line as the last row and sizes the listing to `height - 4`. Deport, Drive browse, the Drive preset list, pass, and the tunnel route list paint a muted key line under the title or in the footer. The spreadsheet and cyclotree already toggle a shortcut list with `?`.

## Goals / Non-Goals

- Hide shortcut text until `?`.
- Give the listing the row a key line used to take.
- Leave prompts, the cyclotree path footer, and the spreadsheet statistics line in place.
- Leave quit keys as they are. `q` still quits the listing screens, including while the list is open. The spreadsheet still leaves only through `:q!`.

## Decisions

- `?` replaces the listing with the shortcut list. A second `?` or Esc restores the listing. On the navigator, `h` still closes the list too.
- While the list is open, other keys do nothing. `q` and ctrl-c still quit the listing screens.
- Esc still quits deport, Drive, pass, the preset list, and the tunnel route list when the shortcut list is closed.
- The tunnel add form keeps `enter save` and `esc cancel`. A removal keeps the `y or n` question. `?` is not a toggle while those prompts are open.
- Listing room becomes `height - 3` on the navigator and deport, and `height - 4` on Drive browse and pass.

## Risks / Trade-offs

- A key that used to be visible is now behind `?`.
