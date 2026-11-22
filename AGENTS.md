# Working on c3po

c3po is the agent CLI. Apps are top-level commands, `c3po <name>`. Agents use those commands and do not open a screen. The apps are `ast`, `clippy`, `deport`, `gates`, `gcal`, `gdrive`, `gmail`, `idealake`, `looptab`, `notify`, `pass`, `slack`, and `tunnel`. `r2d2` is the terminal program. Running `r2d2` opens a screen menu. The screens are `web`, `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, `looptab`, and `slack`. `r2d2 web` is the ryangerardwilson.com admin screen. There is no `c3po tui` command.

- `ast` is the tree in `ast/`. Its command is `c3po ast`.
- The operator opens a screen with `r2d2`. Running `r2d2` opens the screen menu. Agents do not open a screen. `r2d2 web` is the site admin. The screens in this repo are `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, and `slack`. `c3po looptab` is only the CLI: it lists, registers, removes, and shows jobs files and installs them onto the Looptab scheduler file `~/.config/c3po/looptab/looptab` (`$XDG_CONFIG_HOME/c3po/looptab/looptab` when that is absolute). The Looptab timezone file `config.json` and the jobs files sit in the same directory. The installed `looptab` binary opens `~/.config/looptab/looptab`, and on the workstation `~/.config/looptab` is a relative symlink to `c3po/looptab`, so it reads the file c3po writes. It does not open vim. With no arguments it prints the help. `r2d2 looptab` is implemented in r2d2. It lists the jobs, Enter runs the selected job once through `looptab now`, and `e` opens the jobs file in vim and then calls `looptab.Install`. `looptab.JobsFile` and `looptab.Install` are the exported functions it calls; c3po stays the only writer of the scheduler file. `r2d2 slack` shows the Slack desk. `h` and `l` switch workspace. The table is direct messages, tags, and what to know. Enter opens the row. `r` marks it read on the desk. Opening a row does not mark it read. `s` syncs the desk from Slack. The screen also syncs about every 10 minutes. The screen shows when the desk last synced. `xl` is the spreadsheet. It is not a separate command. The trainer is the crusade mode `crusade -m nlp`. Courses stay in the sqlx database `nlp`.
- `c3po idealake` stores one-line ideas of at most 300 characters by topic in the c3po sqlite file. `search` ranks one topic with xAI embeddings. `contains` is the lowercase text match. There is no idealake screen.
- `c3po clippy` copies a multiline text to the Wayland clipboard and does not print it. There is no clippy screen.
- `c3po gates` lists the types, `c3po gates code` lists the statutes, `c3po gates code architecture` lists each provision as its section and heading, and `c3po gates code architecture §1.1` prints that one body of at most 300 characters. A section may also be the bare number, `1.1`, or the old `S.1.1`. Several sections may follow the statute in one command, `c3po gates code architecture 6 7 8.3`, which prints each section on its own line and then its body, in the order written. Every printed body is followed by `receipt <8 hex>`, a hash of the c3po version, the section, and the body. A section with subsections lists those subsections under the body and then names the bodies not shown with the command that prints them. `c3po gates code architecture all` prints every provision with its receipt. `c3po gates code architecture tally <sheet>` counts a sheet of `§1.2 <receipt> <pass|fail> <site> [files <path>...]` lines against this build and the changed files in `git status`, and exits 2 until every provision is judged and every changed file is claimed. Judge every provision; the first failure is the headline. The agent judges. There is no gates screen.
- `c3po notify` sends an omarchy-shell toast through the `ryan.notify` overlay. `install.sh` installs the plugin from `notify/plugin/`. There is no notify screen.
- `deport` frees a safe external port, 8000 through 8999. The screen is `r2d2 deport`. `gcal` is Google Calendar. `gdrive` syncs Google Drive. `gmail` is the Gmail desk. `c3po pass` runs `nebula crypt`. The item screen is `r2d2 pass`, and it reaches 1Password through `nebula crypt`. `tunnel` runs `nebula tunnel`. The route screen is `r2d2 tunnel`.
- c3po is the foundation and the source of truth for each operation. r2d2 is the terminal program above it: an r2d2 screen opens the screen and calls exported c3po functions in process. It does not shell out to `c3po` and does not keep a second copy of an operation. A c3po app prints, records, and installs; it does not open vim, a screen, or any other terminal UI. c3po does not import r2d2. Config for a c3po operation stays under `~/.config/c3po/<app>/` or the sqlx database `c3po`. A screen r2d2 owns, such as looptab, keeps its place through `screen.OpenSession` and `screen.TakeResume`.
- A new app is a new top-level name, with its own package. A new screen is a name under `r2d2`. The screen entry points in this repo are the `screen` package. There is no `c3po tui` command.

`rgw-ast` on `PATH` is this binary. Nutshell and shadow-vim launch that name. Do not install `cx` or `complexity`. `vibetree`, `hide`, `iha`, and `visualize` are not commands.

Verification:

```sh
GOWORK=off go test ./...
./install.sh
c3po version
```

Do not add this module to the home `go.work`. Work and commit inside this Git root.

<!-- rgw-ast:begin -->
# c3po ast boundary

Run `c3po ast status --json` before the first edit in a repository. When it reports
`"enforced": true`, the `c3po ast hook` PreToolUse hook enforces the rules below. Every
change is checked against the file's hash, so concurrent agents and stale context
cannot silently overwrite a file.

- **Trace** code before editing it: `c3po ast callers|callees|impact <symbol>`, then
  `map` and `show`. `search` is for a literal you already know. `c3po ast help callers`
  explains confidence levels and partial results.
- **Read** with `c3po ast read <file> --lines A-B` (or a host Read, `limit` <= `max_read_lines`);
  its `sha256 <hex>` header is the next `--expect-hash`. `c3po ast hash <file>` prints it too.
- **Edit** with `c3po ast patch <file> --expect-hash <sha> --old <text> --new <text>`, or with
  `--old-file <f>`, `--new-file <f>`, `--stdin` (quote the heredoc: `<<'EOF'`). Each patch prints the
  next hash. Relative paths start at the git root. Plain patch is byte-exact; `gofmt -w` is denied.
  To format or edit several files: `c3po ast apply --from-file <manifest.json|git diff>`.
  Go, TypeScript, and Python (`.go` `.ts` `.tsx` `.py`) are refused at `max_cc` or above
  on create, patch, apply, append, and a new-file Write. A function already over that line stays
  editable (default 30). Unparsed Go is refused. `c3po ast gap` lists them.
- **Create** with a host Write to a new path, or `c3po ast create <file> --expect-absent --stdin`.
- **Move** with `c3po ast move <src> <dst> --expect-hash <sha>`; **delete** with
  `c3po ast delete <file> --expect-hash <sha>`.
- **Generators** listed in `generators.allow` (for example `openspec new|archive`) run as
  `c3po ast exec -- <command>`.
- A denied tool call's reason names the exact command to run instead.

Unenforced repositories use normal tools. Policy is global
(`~/.config/c3po/ast/config.toml`). A nested AGENTS.md does not need to repeat this block.
<!-- rgw-ast:end -->
