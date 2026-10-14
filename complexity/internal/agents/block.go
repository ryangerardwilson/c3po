package agents

// Recheck is the command an agent runs before it finishes a turn.
const Recheck = "complexity check --diff HEAD --ratchet --format json --hotspots-only"

// Block is the managed AGENTS.md section.
const Block = `<!-- complexity:begin -->
# complexity boundary

` + "`complexity`" + ` (alias ` + "`cx`" + `) is the global complexity gate. Policy is
` + "`$XDG_CONFIG_HOME/complexity/settings.json`" + `, or
` + "`~/.config/complexity/settings.json`" + ` when that variable is unset. Project
` + "`complexity.toml`" + ` or ` + "`.complexity.toml`" + ` overrides it. CLI flags override both.

Run ` + "`cx budget`" + ` on the files about to change. A function already at its
ceiling needs a new branch in a helper. ` + "`rgw-ast create`" + `, ` + "`rgw-ast patch`" + `, and
` + "`rgw-ast apply`" + ` print a cx note for the source files they wrote. The note
says what became new or worse, the score that passes, and where the points
came from. A legacy function may stay at its previous score. Stop is a
backstop: it reports that note and blocks only when this session made
something new and over a limit or worse. It does not ask you to rewrite a
legacy function down to the limit.

- Keep cognitive complexity ≤ 15, nesting ≤ 4, cyclomatic complexity ≤ 10,
  function length ≤ 80 non-comment lines, and parameters ≤ 6, unless the
  config says otherwise.
- Prefer extracting a nested branch over adding another level.
- Name tables ` + "`<model>_<feature>[_<sub_feature>][_logs]`" + `.
- Order columns: primary key, relationship ids, feature columns, ` + "`data`" + `,
  timestamps.
- Before you finish, run ` + "`complexity check --diff HEAD --ratchet --format json --hotspots-only`" + `.
- A legacy hotspot may stay if you do not make its metrics worse.
- Record a score that may stay with ` + "`cx accept <file> <function> --reason \"...\"`" + `.
- Tests, vendored code, and generated files are excluded by default.
  ` + "`--include-tests`" + ` analyzes tests.

` + "`complexity help`" + ` lists commands. A nested AGENTS.md does not need to repeat
this block.
<!-- complexity:end -->
`
