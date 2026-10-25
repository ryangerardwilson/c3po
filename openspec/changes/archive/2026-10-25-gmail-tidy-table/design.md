## Context

Tidy draws one bar for the whole run. The live line ends with the preset in progress, and the finished line ends with every preset's email and trashed count.

## Goals / Non-Goals

- Goals: a bar with no account name, then a table of preset, email, and trashed count.
- Non-Goals: changing which messages are trashed, the audit list, or the spam-clean bars.

## Decisions

- While the bar runs it shows blocks and done/total. It does not name a preset or an email.
- The finished bar line stays. The table is printed under it, one row per preset, with a header and ruled columns. The trashed count is right-aligned.
- A pipe gets the finished bar, the table, and the completion line.
- No sender groups means no bar and no table.

## Risks / Trade-offs

- A long email widens the table. The columns size to the text.

## Migration Plan

Run `c3po gmail tidy`. The bar moves on its own. The accounts appear in the table when the run finishes, then the completion summary.

## Open Questions

None.
