## Why

The plain Go patch gate keeps a function from rising to `max_cc` or above,
but `patch --symbol`, `create`, and `apply` still write any Go file, so a
restructure through them can add complexity debt the gate was meant to stop.
There is also no command that lists the debt that already exists: the
functions a from-the-start, patch-only history would not hold.

## What Changes

- `patch --symbol`, `create`, and every Go file inside `apply` use the plain
  patch refusal rule on the bytes they would write, after formatting when the
  path formats. A refusal writes nothing (a whole `apply` transaction writes
  nothing, `--dry-run` included), prints one `refused` line per declaration
  sorted by path then name, has no `Error:` line, and exits 1. Successful
  output is unchanged and has no `cc` line. The grandfather rule stays: a
  function already at or above `max_cc` may rise, and a removal never
  refuses. An unclean Go parse is written. Non-Go files are not scored.
- New verb `c3po ast gap [--json] [--check]` lists every scored function at
  or above `max_cc` across the measure scope, in every language cyclotree
  scores, sorted by cyclomatic number descending, then path and name.
  `--check` exits 1 when the list is not empty.
- The scorer for one file is exported from `cyclotree/score`, which package
  cyclotree also uses. It is not in package cyclotree itself, which imports
  ast, so ast could not import it without a cycle.
- The usage catalog gains `ast gap`; help, README, and the agents block name
  the gate on every Go write path and `c3po ast gap`.

## Non-goals

- No gate on `append`, `move`, or `delete`, and no gate for languages other
  than Go.
- No rewrite or split of a function, no CRAP, coverage, duplication, or
  mutation scores, no snapshots, and no new config key.
- No change to the plain patch `cc` lines or to cyclotree's numbers.

## Impact

- Specs: file-boundary, cli-surface, complexity-gap (new).
- Code: `ast/internal/cli`, `ast/internal/files`, `ast/internal/agents`,
  `cyclotree/score`, `cyclotree`, `internal/usage`.
