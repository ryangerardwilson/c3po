## Why

Agents edit Go through `c3po ast patch`. Nothing stops an edit from pushing a
function past the cyclomatic number cyclotree flags, and the agent does not
see what its edit did to complexity. The patch is the one place every agent
edit passes through, so it is where the number can be shown and the line held.

## What Changes

- A plain `c3po ast patch` of a Go file scores every function declaration in
  the old and new bytes with the same cyclomatic number cyclotree reports,
  and prints a `cc` line after the ok line for each declaration whose score
  changed, that is new, or that is gone.
- The patch is refused, nothing is written, and the exit is 1 when a
  declaration rises from below `max_cc` to `max_cc` or above, or a new
  declaration is at `max_cc` or above. A declaration already at or above
  `max_cc` may rise. A removal is never refused.
- Old or new bytes that go/parser rejects, or that have two declarations with
  one name, are written as today with a `cc  <path>  unparsed` line.
- Global config gains `max_cc`, default 30 when missing or not positive. The
  default is applied in memory; the operator's file is not rewritten.
- One counter in `internal/gocyclo` serves both the patch gate and cyclotree,
  so the two cannot disagree. ast does not import cyclotree.

## Non-goals

- No gate on `patch --symbol`, `apply`, `create`, or `append`.
- No gate for languages other than Go.
- No change to cyclotree cognitive or nesting scores, or to its cyclomatic
  rules.
- No change to the ok line format.

## Impact

- Specs: file-boundary, global-config.
- Code: `internal/gocyclo`, `cyclotree/internal/parse`, `ast/internal/cli`,
  `ast/internal/files`, `ast/internal/config`, `ast/internal/agents`.
