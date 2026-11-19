## 1. Hook

- [x] 1.1 Record `git worktree add` and its effective directory in the shell analysis, honouring `git -C`, `cd`, and `sh -c`/`bash -c`
- [x] 1.2 Record `c3po mess tidy add` and its directory
- [x] 1.3 Deny both before the enforcement checks, with the clean check from `mess`

## 2. Check

- [x] 2.1 Test the deny of `git worktree add`, `git -C`, and `bash -c`, enforced or not
- [x] 2.2 Test that `git worktree list`, `git worktree remove`, `c3po mess`, and `c3po mess tidy add` with nothing clean are allowed
- [x] 2.3 Test that `c3po mess tidy add` is denied while a clean marked worktree exists
- [x] 2.4 Validate with `openspec validate deny-worktree-add --strict --no-interactive` and archive it
