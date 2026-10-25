package cli

import "io"

const helpText = `c3po ast — global AST-aware agent boundary

Usage:
  c3po ast [--root <path>] <command> [args]

Commands:
  help [command] | version [--json] (or --version) | config
  status [--json] [--refresh]
  measure [--json] [--refresh]
  doctor [--json] [--refresh]
  agents-block
  map [--json] [--limit N] [--offset N] [--depth N] [path]
  show <symbol|path:symbol|path>
  search [--path <dir>] [--glob <pat>] [--offset N] [--regex] [-i] <query>
  search --help
  callers [--json] [--limit N] [--offset N] <symbol>
  callees [--json] [--limit N] [--offset N] <symbol>
  impact [--json] [--limit N] [--offset N] <symbol>
  explain <path>
  hash <file>...
  read <file> --lines <a>-<b> [--number] [--strict-lines]
  create <file> --expect-absent (--from-file <f>|--stdin) [--parents]
  append <file> --expect-hash <sha> (--from-file <f>|--stdin)
  patch <file> --expect-hash <sha> [--symbol Name | --all] (--old/--new | --old-file/--new-file | --ops-file <json>)
  apply --from-file <manifest.json|diff> [--dry-run] [--no-format]
  delete <file> --expect-hash <sha> [--prune-empty]
  move <src> <dst> --expect-hash <sha> [--parents]
  exec [--json] -- <generator command...>
  hook
  packages [--json]
  affected [--json] [path...]
  graph [--json]
  neighbors [--json] <id>
  cut [--json] [path]

Policy: ~/.config/c3po/ast/config.toml only.
Every walk honors .gitignore, stops at nested git repos and worktrees, and skips
directories that ignore themselves. Prose (Markdown, text) is prose_loc and does
not reach threshold_loc unless enforcement.count_prose is set.
LOC cached ~60s; mutations/hook refresh enforcement. generators.allow lists exec.
Absolute paths outside the current root use their own repository's root.

Outcome tracing (callers, callees, impact) is not capped at the search limit.
Confidence is statically-confirmed, framework-inferred, configuration-inferred,
interface-inferred, or unresolved. truncated=true is not the whole graph.
graph, neighbors, and cut are package structure, not function tracing.
doctor ok is false when enforced and the hook or agents block is not ready.

When enforced:
  trace the outcome with callers/callees/impact, then map/show/search;
  read --lines (or host Read with limit <= max_read_lines);
  create/append/patch/apply/move/delete for mutations
  (host Write of a new file is allowed: it is a create);
  generators only via: c3po ast exec -- <cmd>
  Hosts: pipe PreToolUse to c3po ast hook. Deny reasons give the exact command.

Exit: 0 ok, 1 error, 2 usage (hook always 0 after decision JSON)
Run: c3po ast help <command>
`

const searchHelp = `c3po ast search — literal or regex line search

Usage:
  c3po ast search [--path <dir>] [--glob <pattern>] [--offset N] [--regex] [-i] <query>
  c3po ast search --help

Flags:
  --path          limit walk to this workspace-relative directory or file
  --glob          match basenames or paths (e.g. *.sh)
  --offset N      skip the first N hits (page with the "next" value)
  --regex, -E     the query is a Go regular expression
  -i              case-insensitive

Output is path:line:text, one page of max_search_hits (default 50).
This is not an outcome trace. When more hits exist, stderr prints
"truncated: hits A-B of TOTAL; next: --offset B".
No hits prints a hint on stderr and still exits 0.
A query that looks like a flag is a usage error (exit 2).

Example: c3po ast search --path internal --glob '*.go' HashFile
`

func WriteHelp(w io.Writer) {
	_, _ = io.WriteString(w, helpText)
}

func WriteSearchHelp(w io.Writer) {
	_, _ = io.WriteString(w, searchHelp)
}

const traceHelp = `c3po ast callers, callees, impact — trace what drives an outcome

Usage:
  c3po ast callers [--json] [--limit N] [--offset N] <symbol>
  c3po ast callees [--json] [--limit N] [--offset N] <symbol>
  c3po ast impact [--json] [--limit N] [--offset N] <symbol>

<symbol> is a name, Type.Method, or path:Symbol.
One symbol may return more than the literal search cap.
confidence:
  statically-confirmed     a direct, import, or stored method-value bind
  framework-inferred       route, dependency injection, or callback
  configuration-inferred   router include or registry
  interface-inferred       Go interface dispatch, including implementors
  unresolved               the call was seen and not bound, or the enclosing function is not known
The symbol on an edge is the enclosing function. A nearby const is not that function.
impact returns upstream entrypoint paths and downstream outcome paths.
Each result has assessment.conclusion "complete" or "partial".
Partial means an unresolved edge, a cycle cut, or a depth or path cap.
max_depth is 6 and path_limit is 8. An empty callers or callees array means
none were found. null means that side was not requested.
Do not treat a partial result as the whole outcome.
truncated=true means pass --offset next_offset for the edge list.
Exit 1 when the symbol is missing. Ambiguous names list candidates and do not pick one.

Example: c3po ast impact --json --limit 20 Screen.paint
`

func WriteTraceHelp(w io.Writer) {
	_, _ = io.WriteString(w, traceHelp)
}

// HelpText is the ast overview, or one command. The overview uses an empty name.
func HelpText(cmd string) (string, bool) {
	switch cmd {
	case "":
		return helpText, true
	case "search":
		return searchHelp, true
	case "callers", "callees", "impact":
		return traceHelp, true
	default:
		text, ok := commandHelp[cmd]
		return text, ok
	}
}

func WriteCommandHelp(w io.Writer, cmd string) {
	if cmd == "callers" || cmd == "callees" || cmd == "impact" {
		WriteTraceHelp(w)
		return
	}
	text, ok := commandHelp[cmd]
	if !ok {
		_, _ = io.WriteString(w, "No detailed help for "+cmd+". Run: c3po ast help\n")
		return
	}
	_, _ = io.WriteString(w, text)
}

var commandHelp = map[string]string{
	"map": `c3po ast map [--json] [--limit N] [--offset N] [--depth N] [path]

List symbols in path order, then line order. This is not ranked by graph degree.
--limit defaults to max_map_entries (200). --offset continues a truncated page.
--depth N descends N directory levels; 0 lists only the selected directory.
JSON is {"entries","truncated","next_offset"}. Text ends with
"truncated: true next_offset: N" when another entry exists.
A missing path exits 1. An unknown flag exits 2.

Example: c3po ast map --json --limit 50 --offset 50 internal
`,
	"show": `c3po ast show <symbol|path:symbol|path>

Print one symbol body. Methods use Type.Method or path:Type.Method.
The same symbol identity as map, callers, callees, and impact.
A bare path to a file or directory prints its map outline instead.
Exit 1 when the name is missing or ambiguous.

Example: c3po ast show path/file.py:Screen.paint
`,
	"patch": `c3po ast patch <file> --expect-hash <sha> [--symbol Name | --all] (--old <text> --new <text> | --old-file <f> --new-file <f> | --ops-file <json>)

Exact text swap after the hash matches. Without --symbol the old text must occur once and the bytes are not formatted.
A text that occurs more than once fails and lists every matching line; --all replaces every occurrence.
--ops-file is a JSON array of {"old","new","all"} applied in order, one write.
Each success prints the new hash: pass it as --expect-hash to the next patch of that file.
With --symbol the old text must occur once inside that function, method, or JSX component, and the checkout formatter runs on that file only (gofmt, ruff when declared, biome when biome.json exists, prettier or dprint when that Markdown tool is declared).
A formatter wrapping change does not reject the edit. A missing formatter keeps the edit.
Hash mismatch or a missing symbol exits 1. Missing flags exit 2.

Example: c3po ast patch file.go --expect-hash <sha> --symbol A --old 'old' --new 'new'
`,
	"apply": `c3po ast apply --from-file <manifest.json|diff> [--dry-run] [--no-format]

One transaction for edit, create, delete, and move. Every hash and match is checked before any write. A bad hash, a missing match, or a failed write leaves the tree untouched.
Go, Python, TypeScript, and declared-Markdown files are formatted unless --no-format. Output names the formatter, or "none".
--dry-run prints the planned op and the final unified diff and writes nothing.

JSON manifest: {"files":[{"path","expect_hash","op":"edit|create|delete|move","old","new","symbol","diff","to"}]}
op defaults to edit. create uses expect_hash "absent" or an empty hash. delete takes only the hash. move takes "to" and the hash.
A diff is either a plain unified diff (git diff, diff -u: paths from the ---/+++ headers,
/dev/null for create or delete) or a bundle whose files start with "# path:", "# expect-hash:", "# op:".
A diff edit without a hash is guarded by its context lines.

Example: c3po ast apply --from-file changes.json --dry-run
`,
	"status": `c3po ast status [--json] [--refresh]

Report enforcement plus the selected worktree: branch, HEAD, upstream, dirty, detached, behind, and main worktree path.
--root selects that checkout, including a nested worktree you name. Nested repos are not scanned unless named.
Exit 1 when the root cannot be resolved.

Example: c3po ast --root Work/wiom status --json
`,
	"doctor": `c3po ast doctor [--json] [--refresh]

Diagnostics plus enforcement readiness. ok is true only when there is no hard error and enforcement is not degraded.
enforcement_status is inactive (not enforced), ready (enforced, hook configured, agents block current), or degraded.
ready is false when degraded. hook_ready and agents_block_ready stay separate fields.
A degraded enforced checkout exits 1. That does not waive the command boundary.

Example: c3po ast doctor --json
`,
	"version": `c3po ast version [--json]

Print the stamped version. A dirty source tree keeps the last commit in the version string.
Exit 0.

Example: c3po ast version
`,
	"config": `c3po ast config

Print the global config path and the loaded policy. There is no project-local policy file.
Exit 1 when the global config cannot be read.

Example: c3po ast config
`,
	"measure": `c3po ast measure [--json] [--refresh]

Count lines and files under the selected root. Honors .gitignore and skips nested git repos.
prose_loc counts prose globs (Markdown, text); it reaches loc only when enforcement.count_prose is set.
The cache is about 60 seconds unless --refresh is set.
Exit 1 when the walk fails.

Example: c3po ast measure --json --refresh
`,
	"agents-block": `c3po ast agents-block

Print the canonical AGENTS.md block. This command does not write a file.
Paste it between <!-- c3po ast:begin --> and <!-- c3po ast:end -->.
Exit 0. Extra arguments are a usage error.

Example: c3po ast agents-block
`,
	"explain": `c3po ast explain <path>

Say why a path is included, skipped, or unmapped. A directory reports visible_files, the count map and search see under it.
Exit 1 when the path cannot be explained. Exit 2 when the path is missing from the command.

Example: c3po ast explain internal/trace/trace.go
`,
	"hash": `c3po ast hash <file>...

Print the sha256 of each regular file. Use it immediately before patch, apply, or delete.
Exit 1 when a file is missing. Exit 2 when no file is given.

Example: c3po ast hash internal/cli/help.go
`,
	"read": `c3po ast read <file> --lines <start>-<end> [--number] [--strict-lines]

Print an inclusive line range. While enforced, a whole-file read is denied.
The header is "# <file> lines A-B of TOTAL"; a range over max_read_lines is capped and the header names the next range.
--strict-lines fails when the range is over max_read_lines or passes the end of the file.
Exit 1 on a denied or missing read. Exit 2 on bad flags.

Example: c3po ast read internal/cli/help.go --lines 1-40
`,
	"create": `c3po ast create <file> --expect-absent (--from-file <f>|--stdin) [--parents]

Create one file that must not already exist. --parents creates ancestor directories.
Exit 1 when the path exists or the parent is missing. Exit 2 on bad flags.
For several files, use apply.

Example: c3po ast create notes.md --expect-absent --from-file /tmp/notes.md
`,
	"append": `c3po ast append <file> --expect-hash <sha> (--from-file <f>|--stdin)

Append bytes after the hash matches. The hash is of the file before the append.
Exit 1 on a hash mismatch. Exit 2 on bad flags.

Example: c3po ast append log.txt --expect-hash <sha> --from-file /tmp/more.txt
`,
	"move": `c3po ast move <src> <dst> --expect-hash <sha> [--parents]

Move or rename one verified regular file to a path that does not exist. A git-tracked file moves with git mv, so the rename is staged.
--parents creates the destination directory. For several files, use apply with "op":"move".
Exit 1 on a hash mismatch or an existing destination. Exit 2 on bad flags.

Example: c3po ast move old/name.go new/name.go --expect-hash <sha> --parents
`,
	"delete": `c3po ast delete <file> --expect-hash <sha> [--prune-empty]

Delete one verified regular file. --prune-empty removes empty ancestor directories under the root.
Exit 1 on a hash mismatch. Exit 2 on bad flags.
For several files, use apply.

Example: c3po ast delete tmp.go --expect-hash <sha>
`,
	"exec": `c3po ast exec [--json] -- <command...>

Run one allowlisted generator. generators.allow is global. The report lists created and modified paths with hashes.
Exit 1 when the generator exits non-zero or the command is not allowed. Exit 2 when -- is missing.

Example: c3po ast exec -- npm exec -- openspec validate --all --strict --no-interactive
`,
	"hook": `c3po ast hook

Read one PreToolUse event on stdin and print a permission decision as JSON.
File tools and shell writes are judged by their target path: outside every git
worktree or git-ignored is allowed, otherwise the worktree's enforcement decides.
In an enforced worktree:
  Write of a missing file    allowed (it is a create)
  Write/Edit of a file       denied; the reason gives the patch command and current hash
  Read                       allowed with limit <= max_read_lines or for a short file
  shell writes               denied (redirects, rm/mv/cp, sed -i, git mv, inline
                             python -c/node -e writes); reads are not policed
  generators.allow commands  denied unless run as c3po ast exec -- <cmd>
                             (openspec is denied only for new/archive/init/update)
The event cwd is used unless --root is given.
This command exits 0 after the decision, including a deny.
It does not apply the mutation itself.

Example: c3po ast hook < event.json
`,
	"packages": `c3po ast packages [--json]

Workspace cells: apps, tests, uv members, go.work modules, and other go.mod directories.
A Go module is not given a dependency on a Python package. An empty workspace is [] not null.
Exit 1 when discovery fails.

Example: c3po ast packages --json
`,
	"affected": `c3po ast affected [--json] [path...]

Map paths onto packages. With no paths, use git changes versus HEAD.
Unmapped paths are listed. Exit 0 when nothing is affected.
Exit 1 when discovery or git fails.

Example: c3po ast affected --json internal/trace/trace.go
`,
	"graph": `c3po ast graph [--json]

Package and import edges. Not function tracing. Independent Go and Python cells are not linked.
--json returns nodes and edges arrays. node_count, edge_count, relations, and
component_count match those arrays. degree equals the neighbor list length.
Exit 1 when the graph cannot be built or the counts disagree.

Example: c3po ast graph --json
`,
	"neighbors": `c3po ast neighbors [--json] <id>

Packages or files adjacent to one graph id. Not a caller list.
Exit 1 when the id is unknown. Exit 2 when the id is missing.

Example: c3po ast neighbors --json package:packages/promise-engine
`,
	"cut": `c3po ast cut [--json] [path]

Show the graph cut around a path. Not an outcome trace.
Exit 1 when the path is outside the graph.

Example: c3po ast cut --json packages/promise-engine/engine.py
`,
}
