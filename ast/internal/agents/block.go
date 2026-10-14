package agents

// Block is the canonical managed AGENTS.md boundary for global c3po ast policy.
// It creates no project-local config; hosts embed this text into AGENTS.md.
// Keep it short: trace and hook details live in `c3po scout ast help`.
const Block = `<!-- rgw-ast:begin -->
# c3po ast boundary

Run ` + "`c3po scout ast status --json`" + ` before the first edit in a repository. When it reports
` + "`\"enforced\": true`" + `, the ` + "`c3po scout ast hook`" + ` PreToolUse hook enforces the rules below. Every
change is checked against the file's hash, so concurrent agents and stale context
cannot silently overwrite a file.

- **Trace** code before editing it: ` + "`c3po scout ast callers|callees|impact <symbol>`" + `, then
  ` + "`map`" + ` and ` + "`show`" + `. ` + "`search`" + ` is for a literal you already know. ` + "`c3po scout ast help callers`" + `
  explains confidence levels and partial results.
- **Read** with ` + "`c3po scout ast read <file> --lines A-B`" + `, or a host Read with ` + "`limit`" + ` of at
  most ` + "`max_read_lines`" + ` (200 by default).
- **Edit** with ` + "`c3po scout ast patch <file> --expect-hash <sha> --old <text> --new <text>`" + `.
  ` + "`c3po scout ast hash <file>`" + ` prints the hash; each patch prints the next one. For several
  files at once: ` + "`c3po scout ast apply --from-file <manifest.json|git diff>`" + `.
- **Create** with a host Write to a new path, or ` + "`c3po scout ast create <file> --expect-absent --stdin`" + `.
- **Move** with ` + "`c3po scout ast move <src> <dst> --expect-hash <sha>`" + `; **delete** with
  ` + "`c3po scout ast delete <file> --expect-hash <sha>`" + `.
- **Generators** listed in ` + "`generators.allow`" + ` (for example ` + "`openspec new|archive`" + `) run as
  ` + "`c3po scout ast exec -- <command>`" + `.
- A denied tool call's reason names the exact command to run instead.

Unenforced repositories use normal tools. Policy is global
(` + "`~/.config/rgw-ast/config.toml`" + `). A nested AGENTS.md does not need to repeat this block.
<!-- rgw-ast:end -->
`
