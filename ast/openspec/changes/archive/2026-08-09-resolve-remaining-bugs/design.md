## Decisions

1. Fingerprint `git status --porcelain` + HEAD so nested edits invalidate cache.
2. Mutations and hook always call measure with refresh if near threshold or use
   fresh fingerprint (porcelain already forces invalidate).
3. `exec` snapshots git porcelain before/after, runs command, reports changed
   paths + hashes; requires command matching config generators.allow or
   default openspec/npm openspec patterns.
4. Hook allows shell only if command is `rgw-ast ...` or matches allow after
   being `rgw-ast exec -- ...`.
5. agents-block is a pure renderer of the canonical managed instructions.
6. doctor aggregates config/status/version/hook readiness without writing.
