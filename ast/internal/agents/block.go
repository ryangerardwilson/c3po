package agents

// Block is the canonical managed AGENTS.md boundary for global rgw-ast policy.
// It creates no project-local config; hosts embed this text into AGENTS.md.
// Keep it short: trace and hook details live in `rgw-ast help`.
const Block = `<!-- rgw-ast:begin -->
# rgw-ast boundary

Run ` + "`rgw-ast status --json`" + ` before the first edit in a repository. When it reports
` + "`\"enforced\": true`" + `, the ` + "`rgw-ast hook`" + ` PreToolUse hook enforces the rules below. Every
change is checked against the file's hash, so concurrent agents and stale context
cannot silently overwrite a file.

- **Trace** code before editing it: ` + "`rgw-ast callers|callees|impact <symbol>`" + `, then
  ` + "`map`" + ` and ` + "`show`" + `. ` + "`search`" + ` is for a literal you already know. ` + "`rgw-ast help callers`" + `
  explains confidence levels and partial results.
- **Read** with ` + "`rgw-ast read <file> --lines A-B`" + `, or a host Read with ` + "`limit`" + ` of at
  most ` + "`max_read_lines`" + ` (200 by default).
- **Edit** with ` + "`rgw-ast patch <file> --expect-hash <sha> --old <text> --new <text>`" + `.
  ` + "`rgw-ast hash <file>`" + ` prints the hash; each patch prints the next one. For several
  files at once: ` + "`rgw-ast apply --from-file <manifest.json|git diff>`" + `.
- **Create** with a host Write to a new path, or ` + "`rgw-ast create <file> --expect-absent --stdin`" + `.
- **Move** with ` + "`rgw-ast move <src> <dst> --expect-hash <sha>`" + `; **delete** with
  ` + "`rgw-ast delete <file> --expect-hash <sha>`" + `.
- **Generators** listed in ` + "`generators.allow`" + ` (for example ` + "`openspec new|archive`" + `) run as
  ` + "`rgw-ast exec -- <command>`" + `.
- A denied tool call's reason names the exact command to run instead.

Unenforced repositories use normal tools. Policy is global
(` + "`~/.config/rgw-ast/config.toml`" + `). A nested AGENTS.md does not need to repeat this block.
<!-- rgw-ast:end -->
`
