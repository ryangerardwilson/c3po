## Context

Tidy prints one finished bar per preset. Each line names that account and stays on screen.

## Goals / Non-Goals

- Goals: one bar for the whole tidy run, in-place updates on a terminal, and a finished line that ends with each preset's trashed count.
- Non-Goals: changing which messages are trashed, the audit list, or the spam-clean bars.

## Decisions

- The bar advances across every preset's sender groups. Sixteen cells, with one filled cell once progress has started, and a full bar when the run is done.
- While a preset is in progress, the end of the line names that preset and email. When the run finishes, the line ends with `preset  email  trashed N` for every preset.
- A terminal redraws with carriage return and clear-to-end. Color stays limited to a live terminal with `NO_COLOR` unset and `TERM` other than `dumb`.
- Any other stdout gets the finished line once.
- An error breaks the line before the error text when the bar is on a terminal.
- No senders left after excludes prints no bar. The completion line is the only tidy output.

## Risks / Trade-offs

- The finished status lists every email, so a narrow terminal can wrap that last line. The live line names only the preset in progress, so redraws stay on one row.

## Migration Plan

Run `c3po gmail tidy`. One bar moves across the accounts. The line that remains ends with each account's trashed count, then the completion summary.

## Open Questions

None.
