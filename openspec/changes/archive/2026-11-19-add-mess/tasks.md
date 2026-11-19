## 1. Command

- [x] 1.1 Add `c3po mess` and print the clean, keep, other, and blocked report for the current repository
- [x] 1.2 Add `c3po mess tidy`, `--session`, and `--hook`, removing only clean marked worktrees with `git worktree remove`
- [x] 1.3 Add `c3po mess tidy add` with the `c3po-tidy` marker, refused while a clean entry exists
- [x] 1.4 List `mess` in help, the usage catalog, and the scout spec

## 2. Check

- [x] 2.1 Test the report, the eligibility rules, the marker, removal failure, the session hook, and bad stdin
- [x] 2.2 Test that `c3po` and `c3po help` list mess and `c3po help -f mess` matches `c3po mess help`
- [x] 2.3 Validate this change with `openspec validate add-mess --strict --no-interactive` and archive it
