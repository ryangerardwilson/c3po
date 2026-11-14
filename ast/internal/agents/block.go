package agents

// Block is the canonical managed AGENTS.md boundary for global c3po ast policy.
// It creates no project-local config; hosts embed this text into AGENTS.md.
// Keep it short: trace and hook details live in `c3po ast help`.
const Block = `<!-- rgw-ast:begin -->
# c3po ast boundary

Run ` + "`c3po ast status --json`" + ` before the first edit in a repository. When it reports
` + "`\"enforced\": true`" + `, the ` + "`c3po ast hook`" + ` PreToolUse hook enforces the rules below. Every
change is checked against the file's hash, so concurrent agents and stale context
cannot silently overwrite a file.

- **Trace** code before editing it: ` + "`c3po ast callers|callees|impact <symbol>`" + `, then
  ` + "`map`" + ` and ` + "`show`" + `. ` + "`search`" + ` is for a literal you already know. ` + "`c3po ast help callers`" + `
  explains confidence levels and partial results.
- **Read** with ` + "`c3po ast read <file> --lines A-B`" + ` (or a host Read, ` + "`limit`" + ` <= ` + "`max_read_lines`" + `);
  its ` + "`sha256 <hex>`" + ` header is the next ` + "`--expect-hash`" + `. ` + "`c3po ast hash <file>`" + ` prints it too.
- **Edit** with ` + "`c3po ast patch <file> --expect-hash <sha> --old <text> --new <text>`" + `, or with
  ` + "`--old-file <f>`" + `, ` + "`--new-file <f>`" + `, ` + "`--stdin`" + ` (quote the heredoc: ` + "`<<'EOF'`" + `). Each patch prints the
  next hash. Relative paths start at the git root. Plain patch is byte-exact; ` + "`gofmt -w`" + ` is denied.
  To format or edit several files: ` + "`c3po ast apply --from-file <manifest.json|git diff>`" + `.
  A Go create, patch, symbol patch, or apply that adds a function at ` + "`max_cc`" + ` or above,
  or raises one from below to ` + "`max_cc`" + ` or above, is refused and not written; functions
  already at or above ` + "`max_cc`" + ` stay editable (default 30). ` + "`c3po ast gap`" + ` lists them.
- **Create** with a host Write to a new path, or ` + "`c3po ast create <file> --expect-absent --stdin`" + `.
- **Move** with ` + "`c3po ast move <src> <dst> --expect-hash <sha>`" + `; **delete** with
  ` + "`c3po ast delete <file> --expect-hash <sha>`" + `.
- **Generators** listed in ` + "`generators.allow`" + ` (for example ` + "`openspec new|archive`" + `) run as
  ` + "`c3po ast exec -- <command>`" + `.
- A denied tool call's reason names the exact command to run instead.

Unenforced repositories use normal tools. Policy is global
(` + "`~/.config/c3po/ast/config.toml`" + `). A nested AGENTS.md does not need to repeat this block.
<!-- rgw-ast:end -->
`
